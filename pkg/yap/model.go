package yap

import (
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"mime/multipart"
	"slices"
	"strings"
)

type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

type CreateYapRequest struct {
	Content  string        `json:"content" validate:"required,max=140"`
	MediaIDs []pgtype.UUID `json:"media_ids,omitempty" validate:"omitempty,dive,max=4"`
	Location *Location     `json:"location" validate:"omitempty"`
}

type UpdateYapRequest struct {
	Content  *string        `json:"content" validate:"max=140"`
	MediaIDs []*pgtype.UUID `json:"media_ids,omitempty" validate:"omitempty,dive,max=4"`
	Location *Location      `json:"location" validate:"omitempty"`
}

type MediaItem struct {
	MediaID pgtype.UUID `json:"media_id,omitempty"`
	Type    string      `json:"type" validate:"required,oneof=image video"`
	URL     string      `json:"url" validate:"required,url"`
}

type ListYapsRequest struct {
	UserID string `json:"user_id"`
	Limit  int32  `query:"limit,default=20"`
	Offset int32  `query:"offset,default=0"`
}

type DeleteYapRequest struct {
	YapID pgtype.UUID `json:"yap_id"`
}

type YapResponse struct {
	YapID     pgtype.UUID        `json:"yap_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Content   string             `json:"content"`
	Media     []*MediaItem       `json:"media,omitempty"`
	Hashtags  []string           `json:"hashtags,omitempty"`
	Mentions  []string           `json:"mentions,omitempty"`
	Location  *Location          `json:"location,omitempty"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"edited_at,omitempty" validate:"omitempty"`
}

// ValidateYapContent to validate the content of yap request, it's used for both create and update requests
func (y *CreateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	validateContent(v, y.Content)
	v.Check(len(y.MediaIDs) <= 4, "media", "cannot be more than 4")

	return v.Errors
}

func (y *UpdateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	if y.Content != nil {
		validateContent(v, *y.Content)
	}
	if y.MediaIDs != nil {
		v.Check(len(y.MediaIDs) <= 4, "media", "cannot be more than 4")
	}
	return v.Errors
}

func validateFile(v *util.Validator, file *multipart.FileHeader) map[string]string {
	v.Check(file.Size <= media.MaxFileSize, "media", "must not be greater than 10MB")
	contentType := file.Header.Get("Content-Type")
	v.Check(slices.Contains(media.ValidTypes, contentType), "media", "must contain valid type")
	return v.Errors
}

func validateContent(v *util.Validator, content string) {
	v.Check(len(content) > 0, "content", "must be greater than zero")
	v.Check(len(content) <= 140, "content", "must not be greater than 140")
	v.Check(len(strings.TrimSpace(content)) > 0, "content", "must not be blank")
}
