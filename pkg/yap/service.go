package yap

import (
	"context"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
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
