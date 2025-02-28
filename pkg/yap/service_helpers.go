package yap

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/media"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transaction helper
func executeInTransaction(ctx context.Context, dbpool *pgxpool.Pool, queries *repository.Queries, fn func(repository.Querier, pgx.Tx) error) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := queries.WithTx(tx)

	if err = fn(qtx, tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Content processing helpers
func extractHashtagsAndMentions(content string) ([]string, []string) {
	hashtagRegex := regexp.MustCompile(`#(\w+)`)
	mentionRegex := regexp.MustCompile(`@(\w+)`)

	return extractMatches(hashtagRegex, content), extractMatches(mentionRegex, content)
}

func extractMatches(regex *regexp.Regexp, content string) []string {
	matches := regex.FindAllStringSubmatch(content, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}

// Yap record helpers
func createYapRecord(ctx context.Context, qtx repository.Querier, userID pgtype.UUID,
	req CreateYapRequest, hashtags, mentions []string) (repository.CreateYapRow, error) {

	var lat, lng float64
	if req.Location != nil {
		lat = req.Location.Latitude
		lng = req.Location.Longitude
	}

	return qtx.CreateYap(ctx, repository.CreateYapParams{
		UserID:   userID,
		Content:  req.Content,
		Hashtags: hashtags,
		Mentions: mentions,
		Column5:  lat,
		Column6:  lng,
	})
}

func updateYapRecord(ctx context.Context, qtx repository.Querier,
	yapID, userID pgtype.UUID, req UpdateYapRequest) (repository.UpdateYapRow, error) {

	params, err := buildUpdateParams(req, yapID, userID)
	if err != nil {
		return repository.UpdateYapRow{}, err
	}

	return qtx.UpdateYap(ctx, params)
}

func buildUpdateParams(req UpdateYapRequest, yapID,
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

// Media handling helpers
func associateMedia(ctx context.Context, mediaService media.MediaService, tx pgx.Tx, yapID pgtype.UUID,
	mediaIDs []pgtype.UUID) ([]*MediaItem, error) {

	mediaSvc := mediaService.WithTx(tx)
	mediaItems := make([]*MediaItem, 0, len(mediaIDs))

	for _, mediaID := range mediaIDs {
		mediaDetails, err := mediaSvc.GetUnassociatedMediaByID(ctx, mediaID)
		if err != nil {
			return nil, err
		}

		if err = mediaSvc.UpdateMediaContent(ctx, yapID, mediaID, YapContentType); err != nil {
			return nil, fmt.Errorf("failed to associate media: %w", err)
		}

		mediaItems = append(mediaItems, &MediaItem{
			MediaID: mediaDetails.MediaID,
			Type:    mediaDetails.Type,
			URL:     mediaDetails.Url,
		})
	}

	return mediaItems, nil
}

func handleMediaUpdate(ctx context.Context, mediaService media.MediaService, tx pgx.Tx, yapID pgtype.UUID,
	mediaIDs []*pgtype.UUID) error {

	if mediaIDs == nil {
		return nil
	}

	mediaSvc := mediaService.WithTx(tx)
	if err := mediaSvc.OrphanMedia(ctx, yapID, YapContentType); err != nil {
		return fmt.Errorf("failed to orphan media: %v", err)
	}

	for _, mediaID := range mediaIDs {
		// Check if media exists and unassociated
		_, err := mediaSvc.GetUnassociatedMediaByID(ctx, *mediaID)
		if err != nil {
			return err
		}
		if err = mediaSvc.UpdateMediaContent(ctx, yapID, *mediaID, YapContentType); err != nil {
			return fmt.Errorf("failed to associate media: %v", err)
		}
	}

	return nil
}

// Validation and utility helpers
func validateYap(ctx context.Context, queries *repository.Queries, yapID, userID pgtype.UUID) (bool, error) {
	yap, err := queries.GetYapByID(ctx, yapID)
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

func resolveUserID(ctx context.Context, userIDStr string) (pgtype.UUID, error) {
	var userID pgtype.UUID
	if err := userID.Scan(userIDStr); err != nil {
		return pgtype.UUID{}, apperrors.ErrInvalidUUID
	}

	return userID, nil
}

func buildYapResponses(
	ctx context.Context,
	yaps []repository.ListYapsByUserRow,
	mediaService media.MediaService,
) ([]*YapResponse, error) {

	yapResponses := make([]*YapResponse, len(yaps))
	for i, yap := range yaps {
		yapRow := ConvertGetListYapsByUserRow(yap)
		mediaItems, err := getMediaItems(ctx, mediaService, yap.YapID)
		if err != nil {
			return nil, err
		}
		yapResponses[i] = mapYapToResponse(yapRow, mediaItems)
	}
	return yapResponses, nil
}

func getMediaItems(ctx context.Context, mediaService media.MediaService, yapID pgtype.UUID) ([]*MediaItem, error) {
	mediaDetails, err := mediaService.ListByContent(ctx, yapID, YapContentType)
	if err != nil {
		return nil, err
	}
	mediaItems := make([]*MediaItem, 0, len(mediaDetails))
	for _, item := range mediaDetails {
		mediaItems = append(mediaItems, &MediaItem{
			MediaID: item.MediaID,
			Type:    item.Type,
			URL:     item.Url,
		})

	}

	return mediaItems, nil
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
		UpdatedAt: yap.GetUpdatedAt(),
	}
}
