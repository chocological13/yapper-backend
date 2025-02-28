package yap

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrYapNotFound        = errors.New("yap not found")
	ErrUnauthorizedYapper = errors.New("this yap isn't yours to access")
)

const (
	YapContentType = "yap"
)

// UploadMedia handles media upload for yap related operations
func UploadMedia(ctx context.Context, mediaService media.MediaService, media *multipart.FileHeader) (*MediaItem, error) {
	// Upload to storage
	folderName := fmt.Sprintf("yaps/%s", "DUMMY")
	mediaDetail, err := mediaService.UploadMedia(ctx, media, folderName)
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

func CreateYap(ctx context.Context, dbpool *pgxpool.Pool, mediaService media.MediaService, req CreateYapRequest) (*YapResponse, error) {
	hashtag, mentions := extractHashtagsAndMentions(req.Content)
	queries := repository.New(dbpool)
	var result *YapResponse
	err := executeInTransaction(ctx, dbpool, queries, func(qtx repository.Querier, tx pgx.Tx) error {
		var yap repository.CreateYapRow
		var userUUID pgtype.UUID
		userUUID.Scan("DUMMY")
		yap, err := createYapRecord(ctx, qtx, userUUID, req, hashtag, mentions)
		if err != nil {
			return err
		}

		// Associate media with yap
		var mediaItems []*MediaItem
		mediaItems, err = associateMedia(ctx, mediaService, tx, yap.YapID, req.MediaIDs)

		yapRow := ConvertCreateYapRow(yap)
		result = mapYapToResponse(yapRow, mediaItems)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil

}

func GetYapByID(ctx context.Context, dbpool *pgxpool.Pool, mediaService media.MediaService, yapID pgtype.UUID) (*YapResponse, error) {
	yap, err := repository.New(dbpool).GetYapByID(ctx, yapID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}
		return nil, err
	}

	mediaItems, err := getMediaItems(ctx, mediaService, yapID)
	if err != nil {
		return nil, err
	}

	yapRow := ConvertGetYapByIDRow(yap)
	return mapYapToResponse(yapRow, mediaItems), nil
}

// ListYapsByUser fetches yaps made by a user
func ListYapsByUser(ctx context.Context, dbpool *pgxpool.Pool, mediaService media.MediaService, req ListYapsRequest) ([]*YapResponse, error) {
	userID, err := resolveUserID(ctx, req.UserID)
	println("Here inside ListYapsByUser")
	if err != nil {
		return nil, err
	}

	params := repository.ListYapsByUserParams{
		UserID:  userID,
		Column2: req.Limit,
		Column3: req.Offset,
	}

	yaps, err := repository.New(dbpool).ListYapsByUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrYapNotFound
		}

		return nil, err
	}

	return buildYapResponses(ctx, yaps, mediaService)
}

// UpdateYap updates an existing Yap with the provided information.
// Note: This feature is currently implemented but may be removed in the future
// in the case that a yap is decidedly immutable
func UpdateYap(ctx context.Context, dbpool *pgxpool.Pool, mediaService media.MediaService, yapID pgtype.UUID, req UpdateYapRequest) (*YapResponse, error) {
	var updatedYap repository.UpdateYapRow
	queries := repository.New(dbpool)

	err := executeInTransaction(ctx, dbpool, queries, func(qtx repository.Querier, tx pgx.Tx) error {
		var userID pgtype.UUID
		userID.Scan("DUMMY")
		if _, err := validateYap(ctx, queries, yapID, userID); err != nil {
			return err
		}

		updatedYap, err := updateYapRecord(ctx, qtx, yapID, userID, req)
		if err != nil {
			return err
		}

		err = handleMediaUpdate(ctx, mediaService, tx, updatedYap.YapID, req.MediaIDs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	mediaItems, err := getMediaItems(ctx, mediaService, yapID)
	if err != nil {
		return nil, err
	}

	yapRow := ConvertUpdateYapRow(updatedYap)
	yapResponse := mapYapToResponse(yapRow, mediaItems)

	return yapResponse, nil
}

func DeleteYap(ctx context.Context, dbpool *pgxpool.Pool, mediaService media.MediaService, yapID pgtype.UUID) error {
	queries := repository.New(dbpool)
	return executeInTransaction(ctx, dbpool, queries, func(qtx repository.Querier, tx pgx.Tx) error {
		var userID pgtype.UUID
		userID.Scan("DUMMY")

		yap, err := qtx.GetYapByID(ctx, yapID)
		if err != nil {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				return ErrYapNotFound
			default:
				return err
			}
		}

		if yap.UserID != userID {
			return ErrUnauthorizedYapper
		}

		// Just orphan the media and let cleanup job handle it
		mediaSvc := mediaService.WithTx(tx)
		err = mediaSvc.OrphanMedia(ctx, yap.YapID, YapContentType)
		if err != nil {
			return fmt.Errorf("failed to orphan media: %v", err)
		}

		params := repository.DeleteYapParams{
			YapID:  yapID,
			UserID: userID,
		}

		err = qtx.DeleteYap(ctx, params)
		if err != nil {
			return err
		}

		return err
	})
}
