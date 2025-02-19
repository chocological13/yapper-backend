package yap

import (
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"mime/multipart"
	"slices"
	"strings"
)

type MediaItem struct {
	Type string `json:"type" validate:"required,oneof=image video"`
	URL  string `json:"url" validate:"required,url"`
}

type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

type CreateYapRequest struct {
	Content  string                  `json:"content" validate:"required,max=140"`
	Media    []*multipart.FileHeader `json:"media" validate:"dive,max=4"`
	Location *Location               `json:"location" validate:"omitempty"`
}

type ListYapsRequest struct {
	UserID string `json:"user_id"`
	Limit  int32  `query:"limit,default=20"`
	Offset int32  `query:"offset,default=0"`
}

type UpdateYapRequest struct {
	YapID    pgtype.UUID             `json:"yap_id" validate:"required"`
	Content  *string                 `json:"content,omitempty" validate:"omitempty,max=140"`
	Media    []*multipart.FileHeader `json:"media,omitempty" validate:"omitempty,dive,max=4"`
	Location *Location               `json:"location,omitempty" validate:"omitempty"`
}

type DeleteYapRequest struct {
	YapID pgtype.UUID `json:"yap_id"`
}

type YapResponse struct {
	YapID     pgtype.UUID        `json:"yap_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Content   string             `json:"content"`
	Media     []MediaItem        `json:"media,omitempty"`
	Hashtags  []string           `json:"hashtags,omitempty"`
	Mentions  []string           `json:"mentions,omitempty"`
	Location  *Location          `json:"location,omitempty"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	EditedAt  pgtype.Timestamptz `json:"edited_at,omitempty" validate:"omitempty"`
}

// ValidateYapContent to validate the content of yap request, it's used for both create and update requests
func (input *CreateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	validateContent(v, input.Content)
	validateMedia(v, input.Media)

	return v.Errors
}

func (input *UpdateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	if input.Content != nil {
		validateContent(v, *input.Content)
	}
	if input.Media != nil && len(input.Media) > 0 {
		validateMedia(v, input.Media)
	}

	v.Check(input.YapID.Valid, "yap_id", "must provide yap_id")

	return v.Errors
}

func validateMedia(v *util.Validator, mediaItems []*multipart.FileHeader) {
	v.Check(len(mediaItems) <= 4, "media", "must not be greater than 4")
	for _, file := range mediaItems {
		v.Check(file.Size <= media.MaxFileSize, "media", "must not be greater than 10MB")
		contentType := file.Header.Get("Content-Type")
		v.Check(slices.Contains(media.ValidTypes, contentType), "media", "must contain valid type")
	}
}

func validateContent(v *util.Validator, content string) {
	v.Check(len(content) > 0, "content", "must be greater than zero")
	v.Check(len(content) <= 140, "content", "must not be greater than 140")
	v.Check(len(strings.TrimSpace(content)) > 0, "content", "must not be blank")
}
