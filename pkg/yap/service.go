package yap

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/database"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"mime/multipart"
)

var (
	ErrYapNotFound        = errors.New("yap not found")
	ErrUnauthorizedYapper = errors.New("this yap isn't yours to access")
)

const (
	YapContentType = "yap"
)

type YapService interface {
	UploadMedia(ctx context.Context, media *multipart.FileHeader) (*MediaItem, error)
	CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error)
	GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error)
	ListYapsByUser(ctx context.Context, req ListYapsRequest) ([]*YapResponse, error)
	UpdateYap(ctx context.Context, yapID pgtype.UUID, req UpdateYapRequest) (*YapResponse, error)
	DeleteYap(ctx context.Context, yapID pgtype.UUID) error
}

type yapService struct {
	db           database.PgxPool
	queries      repository.Querier
	userService  users.UserService
	mediaService media.MediaService
}

func NewYapService(db database.PgxPool, queries repository.Querier, userService users.UserService,
	mediaService media.MediaService) YapService {
	return &yapService{
		db:           db,
		queries:      queries,
		userService:  userService,
		mediaService: mediaService}
}

// UploadMedia handles media upload for yap related operations
func (s *yapService) UploadMedia(ctx context.Context, media *multipart.FileHeader) (*MediaItem, error) {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// Upload to storage
	folderName := fmt.Sprintf("yaps/%s", user.ID)
	mediaDetail, err := s.mediaService.UploadMedia(ctx, media, folderName)
	if err != nil {
		return nil, err
	}

	return &MediaItem{
		MediaID: mediaDetail.MediaID,
		Type:    mediaDetail.Type,
		URL:     mediaDetail.Url,
	}, nil
}

// Yap CRUD operations

func (s *yapService) CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error) {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	hashtag, mentions := extractHashtagsAndMentions(req.Content)

	var result *YapResponse
	err = s.executeInTransaction(ctx, func(qtx repository.Querier, tx pgx.Tx) error {
		var yap repository.CreateYapRow
		yap, err = s.createYapRecord(ctx, qtx, user.ID, req, hashtag, mentions)
		if err != nil {
			return err
		}

		// Associate media with yap
		var mediaItems []*MediaItem
		mediaItems, err = s.associateMedia(ctx, tx, yap.YapID, req.MediaIDs)

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
	userID, err := s.resolveUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
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

	return s.buildYapResponses(ctx, yaps)
}

// UpdateYap updates an existing Yap with the provided information.
// Note: This feature is currently implemented but may be removed in the future
// in the case that a yap is decidedly immutable
func (s *yapService) UpdateYap(ctx context.Context, yapID pgtype.UUID, req UpdateYapRequest) (*YapResponse, error) {
	var updatedYap repository.UpdateYapRow

	err := s.executeInTransaction(ctx, func(qtx repository.Querier, tx pgx.Tx) error {
		user, err := s.userService.GetCurrentUser(ctx)
		if err != nil {
			return err
		}

		if _, err = s.validateYap(ctx, yapID, user.ID); err != nil {
			return err
		}

		updatedYap, err = s.updateYapRecord(ctx, qtx, yapID, user.ID, req)
		if err != nil {
			return err
		}

		err = s.handleMediaUpdate(ctx, tx, updatedYap.YapID, req.MediaIDs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	mediaItems, err := s.getMediaItems(ctx, yapID)
	if err != nil {
		return nil, err
	}

	yapRow := ConvertUpdateYapRow(updatedYap)
	yapResponse := mapYapToResponse(yapRow, mediaItems)

	return yapResponse, nil
}

func (s *yapService) DeleteYap(ctx context.Context, yapID pgtype.UUID) error {
	return s.executeInTransaction(ctx, func(qtx repository.Querier, tx pgx.Tx) error {
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
		mediaSvc := s.mediaService.WithTx(tx)
		err = mediaSvc.OrphanMedia(ctx, yap.YapID, YapContentType)
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
