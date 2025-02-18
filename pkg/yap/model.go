package yap

import (
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"net/url"
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
	Content  string      `json:"content" validate:"required,max=140"`
	Media    []MediaItem `json:"media" validate:"dive,max=4"`
	Location *Location   `json:"location" validate:"omitempty"`
}

type YapResponse struct {
	YapID     pgtype.UUID        `json:"yap_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Content   string             `json:"content"`
	Media     []MediaItem        `json:"media"`
	Hashtags  []string           `json:"hashtags"`
	Mentions  []string           `json:"mentions"`
	Location  *Location          `json:"location"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

type ListYapsRequest struct {
	UserID string `json:"user_id"`
	Limit  int32  `query:"limit,default=20"`
	Offset int32  `query:"offset,default=0"`
}

type UpdateYapRequest struct {
	YapID   pgtype.UUID `json:"yap_id" validate:"required"`
	Content string      `json:"content" validate:"required,max=140"`
}

type DeleteYapRequest struct {
	YapID pgtype.UUID `json:"yap_id"`
}

// ValidateYapContent to validate the content of yap request, it's used for both create and update requests
func (input *CreateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	v.Check(len(input.Content) > 0, "content", "must be greater than zero")
	v.Check(len(input.Content) <= 140, "content", "must not be greater than 140")
	v.Check(len(strings.TrimSpace(input.Content)) > 0, "content", "must not be blank")
	v.Check(len(input.Media) <= 4, "media", "must not be greater than 4")

	for _, media := range input.Media {
		v.Check(media.Type == "image" || media.Type == "video", "media_type", "type must be either image or video")

		_, err := url.ParseRequestURI(media.URL)
		v.Check(err == nil, "media_url", "must be a valid URL")
	}

	return v.Errors
}

func (input *UpdateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	v.Check(len(input.Content) > 0, "content", "must be greater than zero")
	v.Check(len(input.Content) <= 140, "content", "must not be greater than 140")
	v.Check(len(strings.TrimSpace(input.Content)) > 0, "content", "must not be blank")
	v.Check(&input.YapID != nil, "yap_id", "must provide yap_id")

	return v.Errors
}
