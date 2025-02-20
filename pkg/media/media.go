package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"mime/multipart"
	"strings"
)

var (
	ErrMediaNotFound = errors.New("media not found")
)

type Service interface {
	UploadMedia(ctx context.Context, file *multipart.FileHeader, folderName string) (*repository.Medium, error)
	GetMediaByID(ctx context.Context, mediaID pgtype.UUID) (*repository.Medium, error)
	GetUnassociatedMediaByID(ctx context.Context, mediaID pgtype.UUID) (*repository.Medium, error)
	ListByContent(ctx context.Context, contentID pgtype.UUID, contentType string) ([]*repository.Medium, error)
	UpdateMediaContent(ctx context.Context, yapID, mediaID pgtype.UUID, contentType string) error
	OrphanMedia(ctx context.Context, mediaID pgtype.UUID, contentType string) error
	DeleteMedia(ctx context.Context, mediaID pgtype.UUID) error
	WithTx(tx pgx.Tx) Service
}

type mediaService struct {
	queries        *repository.Queries
	tx             pgx.Tx
	storageService StorageService
}

func NewService(queries *repository.Queries, storageService StorageService) Service {
	return &mediaService{
		queries:        queries,
		storageService: storageService,
	}
}

func (s *mediaService) WithTx(tx pgx.Tx) Service {
	return &mediaService{
		queries:        repository.New(tx),
		tx:             tx,
		storageService: s.storageService,
	}
}

func (s *mediaService) UploadMedia(ctx context.Context, file *multipart.FileHeader,
	folderName string) (*repository.Medium, error) {
	fileType := file.Header.Get("Content-Type")
	mediaType := determineMediaType(fileType)

	// Upload to storage
	url, err := s.storageService.Upload(ctx, file, folderName)
	if err != nil {
		return nil, err
	}

	// Create media record
	mediaDetail, err := s.queries.CreateMedia(ctx, repository.CreateMediaParams{
		Type: mediaType,
		Url:  url,
	})
	if err != nil {
		_ = s.storageService.Delete(ctx, url) // cleanup on error
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}

	return &mediaDetail, nil
}

func (s *mediaService) GetMediaByID(ctx context.Context, mediaID pgtype.UUID) (*repository.Medium, error) {
	mediaDetail, err := s.getQueries().GetMediaByID(ctx, mediaID)
	if err != nil {
		return nil, ErrMediaNotFound
	}
	return &mediaDetail, nil
}

func (s *mediaService) GetUnassociatedMediaByID(ctx context.Context, mediaID pgtype.UUID) (*repository.Medium, error) {
	mediaDetail, err := s.getQueries().GetUnassociatedMediaByID(ctx, mediaID)
	if err != nil {
		return nil, ErrMediaNotFound
	}
	return &mediaDetail, nil
}

func (s *mediaService) ListByContent(ctx context.Context, contentID pgtype.UUID,
	contentType string) ([]*repository.Medium, error) {
	mediaDetails, err := s.getQueries().GetMediaForContent(ctx, repository.GetMediaForContentParams{
		ContentID: contentID,
		ContentType: pgtype.Text{
			String: contentType,
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
	return createMediaList(mediaDetails), nil
}

func (s *mediaService) UpdateMediaContent(ctx context.Context, yapID, mediaID pgtype.UUID, contentType string) error {
	return s.getQueries().UpdateMediaContent(ctx, repository.UpdateMediaContentParams{
		ContentID: yapID,
		ContentType: pgtype.Text{
			String: contentType,
			Valid:  true,
		},
		MediaID: mediaID,
	})
}

func (s *mediaService) OrphanMedia(ctx context.Context, mediaID pgtype.UUID, contentType string) error {
	return s.getQueries().OrphanMedia(ctx, repository.OrphanMediaParams{
		ContentID: mediaID,
		ContentType: pgtype.Text{
			String: contentType,
			Valid:  true,
		},
	})
}

func (s *mediaService) DeleteMedia(ctx context.Context, mediaID pgtype.UUID) error {
	return s.getQueries().DeleteMedia(ctx, mediaID)
}

// -------------------- 🔽 HELPER FUNCTIONS BELOW 🔽 --------------------

func (s *mediaService) getQueries() *repository.Queries {
	if s.tx != nil {
		return repository.New(s.tx)
	}
	return s.queries
}

func determineMediaType(contentType string) string {
	if strings.HasPrefix(contentType, "video/") {
		return "video"
	}
	return "image"
}

func createMediaList(mediaDetails []repository.Medium) []*repository.Medium {
	mediaList := make([]*repository.Medium, 0, len(mediaDetails))
	for i := range mediaDetails {
		mediaList = append(mediaList, &mediaDetails[i])
	}
	return mediaList
}
