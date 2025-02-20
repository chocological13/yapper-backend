package yap

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Transaction helper
func (s *yapService) executeInTransaction(ctx context.Context, fn func(*repository.Queries, pgx.Tx) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

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
func (s *yapService) createYapRecord(ctx context.Context, qtx *repository.Queries, userID pgtype.UUID,
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

func (s *yapService) updateYapRecord(ctx context.Context, qtx *repository.Queries,
	yapID, userID pgtype.UUID, req UpdateYapRequest) (repository.UpdateYapRow, error) {

	params, err := s.buildUpdateParams(req, yapID, userID)
	if err != nil {
		return repository.UpdateYapRow{}, err
	}

	return qtx.UpdateYap(ctx, params)
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

// Media handling helpers
func (s *yapService) associateMedia(ctx context.Context, tx pgx.Tx, yapID pgtype.UUID,
	mediaIDs []pgtype.UUID) ([]*MediaItem, error) {

	mediaSvc := s.mediaService.WithTx(tx)
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

func (s *yapService) handleMediaUpdate(ctx context.Context, tx pgx.Tx, yapID pgtype.UUID,
	mediaIDs []*pgtype.UUID) error {

	if mediaIDs == nil {
		return nil
	}

	mediaSvc := s.mediaService.WithTx(tx)
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

func (s *yapService) resolveUserID(ctx context.Context, userIDStr string) (pgtype.UUID, error) {
	if userIDStr == "" {
		user, err := s.userService.GetCurrentUser(ctx)
		if err != nil {
			return pgtype.UUID{}, err
		}
		return user.ID, nil
	}

	var userID pgtype.UUID
	if err := userID.Scan(userIDStr); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid user id: %s", err)
	}

	return userID, nil
}

func (s *yapService) buildYapResponses(ctx context.Context,
	yaps []repository.ListYapsByUserRow) ([]*YapResponse, error) {

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

func (s *yapService) getMediaItems(ctx context.Context, yapID pgtype.UUID) ([]*MediaItem, error) {
	mediaDetails, err := s.mediaService.ListByContent(ctx, yapID, YapContentType)
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
		EditedAt:  yap.GetUpdatedAt(),
	}
}
