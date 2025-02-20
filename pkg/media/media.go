package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"mime/multipart"
	"runtime"
	"strings"
	"sync"
	"time"
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
	CleanUpOrphanedMedia(ctx context.Context) error
	HardDeleteSoftDeletedMedia(ctx context.Context) error
	WithTx(tx pgx.Tx) Service
}

type mediaService struct {
	db             *pgxpool.Pool
	queries        *repository.Queries
	tx             pgx.Tx
	storageService StorageService
	logger         *slog.Logger
}

func NewService(db *pgxpool.Pool, queries *repository.Queries, storageService StorageService,
	logger *slog.Logger) Service {
	return &mediaService{
		db:             db,
		queries:        queries,
		storageService: storageService,
		logger:         logger,
	}
}

func (s *mediaService) WithTx(tx pgx.Tx) Service {
	return &mediaService{
		db:             s.db,
		queries:        repository.New(tx),
		tx:             tx,
		storageService: s.storageService,
		logger:         s.logger,
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

func (s *mediaService) CleanUpOrphanedMedia(ctx context.Context) error {
	orphaned, err := s.queries.GetOrphanedMedia(ctx, convertTimestamp(time.Now().Add(-6*time.Hour)))
	if err != nil {
		return fmt.Errorf("error getting orphaned media: %w", err)
	}
	s.logger.Info("orphaned media found", "count", len(orphaned))

	var wg sync.WaitGroup
	errChan := make(chan error, len(orphaned))

	sem := make(chan struct{}, runtime.NumCPU())

	for _, media := range orphaned {
		wg.Add(1)

		go func(m repository.Medium) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err = s.cleanupSingleMedia(ctx, media); err != nil {
				errChan <- fmt.Errorf("failed to cleanup media %s: %w", m.MediaID, err)
			}

		}(media)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var errs []error
	for err = range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup completed with errors: %v", errs)
	}

	return nil
}

func (s *mediaService) HardDeleteSoftDeletedMedia(ctx context.Context) error {
	threshold := convertTimestamp(time.Now().Add(-7 * 24 * time.Hour))
	return s.queries.HardDeleteSoftDeletedMedia(ctx, threshold)
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

func convertTimestamp(time time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  time,
		Valid: true,
	}
}

func (s *mediaService) cleanupSingleMedia(ctx context.Context, media repository.Medium) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

	// Attempt to soft delete in DB
	if err = qtx.SoftDeleteMedia(ctx, media.MediaID); err != nil {
		return fmt.Errorf("failed to soft delete media record: %w", err)
	}

	// Attempt to delete from storage
	if err = s.storageService.Delete(ctx, media.Url); err != nil {
		return fmt.Errorf("failed to delete media from storage: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
