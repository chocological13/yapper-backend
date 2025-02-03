package yap

import (
	"context"
	"database/sql"
	"errors"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrYapNotFound = errors.New("Yap not found")
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
		if errors.Is(err, sql.ErrNoRows) {
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
