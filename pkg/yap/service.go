package yap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"regexp"
)

var (
	ErrYapNotFound        = errors.New("Yap not found")
	ErrUnauthorizedYapper = errors.New("This yap isn't yours to access")
)

type Service interface {
	CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error)
	GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error)
	ListYapsByUser(ctx context.Context, req ListYapsRequest) ([]*YapResponse, error)
	UpdateYap(ctx context.Context, req UpdateYapRequest) (*YapResponse, error)
	DeleteYap(ctx context.Context, yapID pgtype.UUID) error
}

type yapService struct {
	queries     *repository.Queries
	userService users.UserService
}

func NewService(queries *repository.Queries, userService users.UserService) Service {
	return &yapService{queries: queries,
		userService: userService}
}

func (s *yapService) CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error) {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	hashtag, mentions := extractHashtagsAndMentions(req.Content)

	mediaJSON, err := json.Marshal(req.Media)
	if err != nil {
		return nil, err
	}

	var lat, lng float64
	if req.Location != nil {
		lat = req.Location.Latitude
		lng = req.Location.Longitude
	}

	yap, err := s.queries.CreateYap(ctx, repository.CreateYapParams{
		UserID:   user.ID,
		Content:  req.Content,
		Media:    mediaJSON,
		Hashtags: hashtag,
		Mentions: mentions,
		Column6:  lat,
		Column7:  lng,
	})

	if err != nil {
		return nil, err
	}

	return mapYapToResponse(yap)
}

func (s *yapService) GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error) {
	yap, err := s.queries.GetYapByID(ctx, yapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	return &YapResponse{
		YapID:     yap.YapID,
		UserID:    yap.UserID,
		Content:   yap.Content,
		CreatedAt: yap.CreatedAt,
	}, nil
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
		yapResponses[i] = &YapResponse{
			YapID:     yap.YapID,
			UserID:    yap.UserID,
			Content:   yap.Content,
			CreatedAt: yap.CreatedAt,
		}
	}

	return yapResponses, nil
}

// UpdateYap updates an existing Yap with the provided information.
// Note: This feature is currently implemented but may be removed in the future
// in the case that a yap is decidedly immutable
func (s *yapService) UpdateYap(ctx context.Context, req UpdateYapRequest) (*YapResponse, error) {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	params := repository.UpdateYapParams{
		YapID:   req.YapID,
		Content: req.Content,
		UserID:  user.ID,
	}

	yap, err := s.queries.GetYapByID(ctx, req.YapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	// check if yap actually belongs to the user
	if yap.UserID != user.ID {
		return nil, ErrUnauthorizedYapper
	}

	yap, err = s.queries.UpdateYap(ctx, params)
	if err != nil {
		return nil, err
	}

	return &YapResponse{
		YapID:     yap.YapID,
		UserID:    yap.UserID,
		Content:   yap.Content,
		CreatedAt: yap.CreatedAt,
	}, nil
}

func (s *yapService) DeleteYap(ctx context.Context, yapID pgtype.UUID) error {
	user, err := s.userService.GetCurrentUser(ctx)
	if err != nil {
		return err
	}

	yap, err := s.GetYapByID(ctx, yapID)
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

	params := repository.DeleteYapParams{
		YapID:  yapID,
		UserID: user.ID,
	}

	err = s.queries.DeleteYap(ctx, params)
	return err
}

func extractHashtagsAndMentions(content string) ([]string, []string) {
	hashtagRegex := regexp.MustCompile(`#(\w+)`)
	mentionRegex := regexp.MustCompile(`@(\w+)`)

	var hashtags, mentions []string

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

func mapYapToResponse(yap repository.CreateYapRow) (*YapResponse, error) {
	var media []MediaItem
	if err := json.Unmarshal(yap.Media, &media); err != nil {
		return nil, err
	}

	var location *Location
	if yap.Longitude != nil && yap.Latitude != nil {
		location = &Location{
			Latitude:  yap.Latitude.(float64),
			Longitude: yap.Longitude.(float64),
		}
	}

	return &YapResponse{
		YapID:     yap.YapID,
		UserID:    yap.UserID,
		Content:   yap.Content,
		Media:     media,
		Hashtags:  yap.Hashtags,
		Mentions:  yap.Mentions,
		Location:  location,
		CreatedAt: yap.CreatedAt,
	}, nil
}
