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

	yapRow := ConvertCreateYapRow(yap)
	return mapYapToResponse(yapRow), nil
}

func (s *yapService) GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error) {
	yap, err := s.queries.GetYapByID(ctx, yapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	yapRow := ConvertGetYapByIDRow(yap)
	return mapYapToResponse(yapRow), nil
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
		yapResponses[i] = mapYapToResponse(yapRow)
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

	if _, err = s.validateYap(ctx, req.YapID, user.ID); err != nil {
		return nil, err
	}

	params, err := s.buildUpdateParams(req, user.ID)
	if err != nil {
		return nil, err
	}

	editedYap, err := s.queries.UpdateYap(ctx, params)
	if err != nil {
		return nil, err
	}

	yapRow := ConvertUpdateYapRow(editedYap)
	return mapYapToResponse(yapRow), nil
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

func mapYapToResponse(yap YapRow) *YapResponse {
	var media []MediaItem
	if err := json.Unmarshal(yap.GetMedia(), &media); err != nil {
		return nil
	}

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
		Media:     media,
		Hashtags:  yap.GetHashtags(),
		Mentions:  yap.GetMentions(),
		Location:  location,
		CreatedAt: yap.GetCreatedAt(),
		EditedAt:  yap.GetUpdatedAt(),
	}
}

func (s *yapService) buildUpdateParams(req UpdateYapRequest, userID pgtype.UUID) (repository.UpdateYapParams, error) {
	params := repository.UpdateYapParams{
		YapID:  req.YapID,
		UserID: userID,
	}

	if req.Content != nil {
		params.Column2 = *req.Content
		if *req.Content != "" {
			hashtags, mentions := extractHashtagsAndMentions(*req.Content)
			params.Column4 = hashtags
			params.Column5 = mentions
		} else {
			params.Column4 = []string{}
			params.Column5 = []string{}
		}
	}

	var err error
	if req.Media != nil {
		var mediaJSON []byte
		mediaJSON, err = json.Marshal(req.Media)
		if err != nil {
			return repository.UpdateYapParams{}, err
		}
		params.Column3 = mediaJSON
	}

	if req.Location != nil {
		if req.Location.Latitude == 0 && req.Location.Longitude == 0 {
			params.Column8 = true
		} else {
			params.Column6 = req.Location.Latitude
			params.Column7 = req.Location.Longitude
			params.Column8 = false
		}
	} else {
		params.Column8 = false
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
