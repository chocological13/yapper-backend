package yap

import (
	"context"
	"errors"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrYapNotFound        = errors.New("Yap not found")
	ErrUnauthorizedYapper = errors.New("This yap isn't yours to access")
)

type Service struct {
	queries *repository.Queries
}

func NewService(queries *repository.Queries) *Service {
	return &Service{queries: queries}
}

// CreateYap handles the creation of a new Yap in the system.
// ! still in progress without getting user info from context
// TODO : Future params would be ctx, userID (from ctx in handler), and req
func (s *Service) CreateYap(ctx context.Context, req CreateYapRequest) (*YapResponse, error) {
	//yap, err := s.queries.CreateYap(ctx, repository.CreateYapParams{
	//	UserID:  userID,
	//	Content: req.Content,
	//})

	yap, err := s.queries.CreateYap(ctx, repository.CreateYapParams{
		UserID:  req.UserID,
		Content: req.Content,
	})

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

func (s *Service) GetYapByID(ctx context.Context, yapID pgtype.UUID) (*YapResponse, error) {
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
func (s *Service) ListYapsByUser(ctx context.Context, userID pgtype.UUID, req ListYapsRequest) ([]*YapResponse, error) {
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
func (s *Service) UpdateYap(ctx context.Context, req UpdateYapRequest) (*YapResponse, error) {
	params := repository.UpdateYapParams{
		YapID:   req.YapID,
		Content: req.Content,
		UserID:  req.UserID,
	}

	yap, err := s.queries.GetYapByID(ctx, req.YapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	// TODO : check if yap actually belongs to the user
	// right now it's checking from request, but check with the logged in user in the future
	if yap.UserID != req.UserID {
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
