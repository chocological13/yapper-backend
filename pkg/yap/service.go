package yap

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mime/multipart"
	"regexp"
	"strings"
)

var (
	ErrYapNotFound        = errors.New("yap not found")
	ErrUnauthorizedYapper = errors.New("this yap isn't yours to access")
)

type Service interface {
	UploadMedia(ctx context.Context, media *multipart.FileHeader) (*MediaItem, error)
	CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error)
	GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error)
	ListYapsByUser(ctx context.Context, req ListYapsRequest) ([]*YapResponse, error)
	UpdateYap(ctx context.Context, yapID pgtype.UUID, req UpdateYapRequest) (*YapResponse, error)
	DeleteYap(ctx context.Context, yapID pgtype.UUID) error
}

type yapService struct {
	db           *pgxpool.Pool
	queries      *repository.Queries
	userService  users.UserService
	mediaService media.Service
}

func NewService(db *pgxpool.Pool, queries *repository.Queries, userService users.UserService,
	mediaService media.Service) Service {
	return &yapService{
		db:           db,
		queries:      queries,
		userService:  userService,
		mediaService: mediaService}
}

func (s *yapService) UploadMedia(ctx context.Context, media *multipart.FileHeader) (*MediaItem, error) {
	contentType := media.Header.Get("Content-Type")
	mediaType := "image"
	if strings.HasPrefix(contentType, "video/") {
		mediaType = "video"
	}

	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// Upload to storage
	folderName := fmt.Sprintf("yaps/%s", user.ID)
	url, err := s.mediaService.UploadMedia(ctx, media, folderName)
	if err != nil {
		return nil, err
	}

	// Create media record
	mediaDetails, err := s.queries.CreateMedia(ctx, repository.CreateMediaParams{
		Type: mediaType,
		Url:  url,
	})
	if err != nil {
		// cleanup the uploaded file
		_ = s.mediaService.DeleteMedia(ctx, url)
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}

	return &MediaItem{
		MediaID: mediaDetails.MediaID,
		Type:    mediaDetails.Type,
		URL:     mediaDetails.Url,
	}, nil
}

func (s *yapService) CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error) {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	hashtag, mentions := extractHashtagsAndMentions(req.Content)

	var result *YapResponse
	err = s.executeInTransaction(ctx, func(qtx *repository.Queries) error {
		var lat, lng float64
		if req.Location != nil {
			lat = req.Location.Latitude
			lng = req.Location.Longitude
		}

		var yap repository.CreateYapRow
		yap, err = qtx.CreateYap(ctx, repository.CreateYapParams{
			UserID:   user.ID,
			Content:  req.Content,
			Hashtags: hashtag,
			Mentions: mentions,
			Column5:  lat,
			Column6:  lng,
		})

		if err != nil {
			return err
		}

		// Associate media with yap
		var mediaItems []*MediaItem
		for _, mediaID := range req.MediaIDs {

			var mediaDetails repository.Medium
			mediaDetails, err = qtx.GetUnassociatedMediaByID(ctx, mediaID)
			if err != nil {
				switch err {
				case pgx.ErrNoRows:
					return media.ErrMediaNotFound
				}
				return err
			}

			err = qtx.UpdateMediaContent(ctx, repository.UpdateMediaContentParams{
				ContentID: yap.YapID,
				ContentType: pgtype.Text{
					String: "yap",
					Valid:  true,
				},
				MediaID: mediaID,
			})
			if err != nil {
				return fmt.Errorf("failed to associate media: %w", err)
			}
			mediaItems = append(mediaItems, &MediaItem{
				MediaID: mediaDetails.MediaID,
				Type:    mediaDetails.Type,
				URL:     mediaDetails.Url,
			})
		}

		yapRow := ConvertCreateYapRow(yap)
		result = mapYapToResponse(yapRow, mediaItems)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil

}

func (s *yapService) GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error) {
	yap, err := s.queries.GetYapByID(ctx, yapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	mediaItems, err := s.getMediaItems(ctx, yapID)
	if err != nil {
		return nil, err
	}

	yapRow := ConvertGetYapByIDRow(yap)
	return mapYapToResponse(yapRow, mediaItems), nil
}

// ListYapsByUser fetches yaps made by a user
func (s *yapService) ListYapsByUser(ctx context.Context, req ListYapsRequest) ([]*YapResponse, error) {
	var userID pgtype.UUID

	if req.UserID == "" {
		user, err := s.userService.GetCurrentUser(ctx)
		if err != nil {
			return nil, err
		}
		userID = user.ID
	} else {
		err := userID.Scan(req.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user id: %s", err)
		}
	}

	params := repository.ListYapsByUserParams{
		UserID:  userID,
		Column2: req.Limit,
		Column3: req.Offset,
	}

	yaps, err := s.queries.ListYapsByUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}

		return nil, err
	}

	yapResponses := make([]*YapResponse, len(yaps))
	for i, yap := range yaps {
		yapRow := ConvertGetListYapsByUserRow(yap)
		mediaItems, err := s.getMediaItems(ctx, yap.YapID)
		if err != nil {
			return nil, err
		}
		yapResponses[i] = mapYapToResponse(yapRow, mediaItems)
	}

	return yapResponses, nil
}

// UpdateYap updates an existing Yap with the provided information.
// Note: This feature is currently implemented but may be removed in the future
// in the case that a yap is decidedly immutable
func (s *yapService) UpdateYap(ctx context.Context, yapID pgtype.UUID, req UpdateYapRequest) (*YapResponse, error) {
	var yapResponse *YapResponse

	err := s.executeInTransaction(ctx, func(qtx *repository.Queries) error {
		user, err := s.userService.GetCurrentUser(ctx)
		if err != nil {
			return err
		}

		if _, err = s.validateYap(ctx, yapID, user.ID); err != nil {
			return err
		}

		params, err := s.buildUpdateParams(req, yapID, user.ID)
		if err != nil {
			return err
		}

		updatedYap, err := s.queries.UpdateYap(ctx, params)
		if err != nil {
			return err
		}

		// Handle media updates if provided
		if req.MediaIDs != nil {
			// Orphan existing media
			err = qtx.OrphanMedia(ctx, repository.OrphanMediaParams{
				ContentID: yapID,
				ContentType: pgtype.Text{
					String: "yap",
					Valid:  true,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to orphan existing media: %v", err)
			}

			// Associate new media
			for _, mediaID := range req.MediaIDs {
				err = qtx.UpdateMediaContent(ctx, repository.UpdateMediaContentParams{
					ContentID: updatedYap.YapID,
					ContentType: pgtype.Text{
						String: "yap",
						Valid:  true,
					},
					MediaID: *mediaID,
				})
				if err != nil {
					return fmt.Errorf("failed to associate media: %v", err)
				}
			}
		}

		mediaDetails, err := qtx.GetMediaForContent(ctx, repository.GetMediaForContentParams{
			ContentID: updatedYap.YapID,
			ContentType: pgtype.Text{
				String: "yap",
				Valid:  true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to associate media: %v", err)
		}

		var mediaItems []*MediaItem
		for _, item := range mediaDetails {
			mediaItems = append(mediaItems, &MediaItem{
				MediaID: item.MediaID,
				Type:    item.Type,
				URL:     item.Url,
			})
		}

		yapRow := ConvertUpdateYapRow(updatedYap)
		yapResponse = mapYapToResponse(yapRow, mediaItems)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return yapResponse, nil
}

func (s *yapService) DeleteYap(ctx context.Context, yapID pgtype.UUID) error {
	return s.executeInTransaction(ctx, func(qtx *repository.Queries) error {
		user, err := s.userService.GetCurrentUser(ctx)
		if err != nil {
			return err
		}

		yap, err := qtx.GetYapByID(ctx, yapID)
		if err != nil {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				return ErrYapNotFound
			default:
				return err
			}
		}

		if yap.UserID != user.ID {
			return ErrUnauthorizedYapper
		}

		// Just orphan the media and let cleanup job handle it
		err = qtx.OrphanMedia(ctx, repository.OrphanMediaParams{
			ContentID: yap.YapID,
			ContentType: pgtype.Text{
				String: "yap",
				Valid:  true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to orphan media: %v", err)
		}

		params := repository.DeleteYapParams{
			YapID:  yapID,
			UserID: user.ID,
		}

		err = qtx.DeleteYap(ctx, params)
		if err != nil {
			return err
		}

		return err
	})
}

func extractHashtagsAndMentions(content string) ([]string, []string) {
	hashtagRegex := regexp.MustCompile(`#(\w+)`)
	mentionRegex := regexp.MustCompile(`@(\w+)`)

	hashtags := []string{}
	mentions := []string{}

	for _, match := range hashtagRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			hashtags = append(hashtags, match[1])
		}
	}

	for _, match := range mentionRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	return hashtags, mentions
}

func mapYapToResponse(yap YapRow, mediaItems []*MediaItem) *YapResponse {
	var location *Location
	if yap.GetLongitude() != nil && yap.GetLatitude() != nil {
		location = &Location{
			Latitude:  yap.GetLatitude().(float64),
			Longitude: yap.GetLongitude().(float64),
		}
	}

	return &YapResponse{
		YapID:     yap.GetYapID(),
		UserID:    yap.GetUserID(),
		Content:   yap.GetContent(),
		Media:     mediaItems,
		Hashtags:  yap.GetHashtags(),
		Mentions:  yap.GetMentions(),
		Location:  location,
		CreatedAt: yap.GetCreatedAt(),
		EditedAt:  yap.GetUpdatedAt(),
	}
}

// executeInTransaction wraps the execution of a function in a database transaction
func (s *yapService) executeInTransaction(ctx context.Context, fn func(*repository.Queries) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

	if err = fn(qtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *yapService) buildUpdateParams(req UpdateYapRequest, yapID,
	userID pgtype.UUID) (repository.UpdateYapParams,
	error) {
	params := repository.UpdateYapParams{
		YapID:  yapID,
		UserID: userID,
	}

	if req.Content != nil {
		params.Column2 = *req.Content
		if *req.Content != "" {
			hashtags, mentions := extractHashtagsAndMentions(*req.Content)
			params.Column3 = hashtags
			params.Column4 = mentions
		} else {
			params.Column3 = []string{}
			params.Column4 = []string{}
		}
	}

	if req.Location != nil {
		if req.Location.Latitude == 0 && req.Location.Longitude == 0 {
			params.Column7 = true
		} else {
			params.Column5 = req.Location.Latitude
			params.Column6 = req.Location.Longitude
			params.Column7 = false
		}
	} else {
		params.Column7 = false
	}

	return params, nil
}

func (s *yapService) validateYap(ctx context.Context, yapID, userID pgtype.UUID) (bool, error) {
	yap, err := s.queries.GetYapByID(ctx, yapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrYapNotFound
		}
		return false, err
	}

	if yap.UserID != userID {
		return false, ErrUnauthorizedYapper
	}

	return true, nil
}

func (s *yapService) getMediaItems(ctx context.Context, yapID pgtype.UUID) ([]*MediaItem, error) {
	var mediaItems []*MediaItem
	mediaDetails, err := s.queries.GetMediaForContent(ctx, repository.GetMediaForContentParams{
		ContentID: yapID,
		ContentType: pgtype.Text{
			String: "yap",
			Valid:  true,
		},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	for _, item := range mediaDetails {
		mediaItems = append(mediaItems, &MediaItem{
			MediaID: item.MediaID,
			Type:    item.Type,
			URL:     item.Url,
		})
	}

	return mediaItems, nil
}
