package yap

import (
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type YapRow interface {
	GetYapID() pgtype.UUID
	GetUserID() pgtype.UUID
	GetContent() string
	GetMedia() []byte
	GetHashtags() []string
	GetMentions() []string
	GetLongitude() interface{}
	GetLatitude() interface{}
	GetCreatedAt() pgtype.Timestamptz
	GetUpdatedAt() pgtype.Timestamptz
}

type Yap struct {
	YapID     pgtype.UUID
	UserID    pgtype.UUID
	Content   string
	Media     []byte
	Hashtags  []string
	Mentions  []string
	Longitude interface{}
	Latitude  interface{}
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

func (y Yap) GetYapID() pgtype.UUID            { return y.YapID }
func (y Yap) GetUserID() pgtype.UUID           { return y.UserID }
func (y Yap) GetContent() string               { return y.Content }
func (y Yap) GetMedia() []byte                 { return y.Media }
func (y Yap) GetHashtags() []string            { return y.Hashtags }
func (y Yap) GetMentions() []string            { return y.Mentions }
func (y Yap) GetLongitude() interface{}        { return y.Longitude }
func (y Yap) GetLatitude() interface{}         { return y.Latitude }
func (y Yap) GetCreatedAt() pgtype.Timestamptz { return y.CreatedAt }
func (y Yap) GetUpdatedAt() pgtype.Timestamptz { return y.UpdatedAt }

func ConvertCreateYapRow(row repository.CreateYapRow) Yap {
	return Yap{
		YapID:     row.YapID,
		UserID:    row.UserID,
		Content:   row.Content,
		Media:     row.Media,
		Hashtags:  row.Hashtags,
		Mentions:  row.Mentions,
		Longitude: row.Longitude,
		Latitude:  row.Latitude,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func ConvertGetYapByIDRow(row repository.GetYapByIDRow) Yap {
	return Yap{
		YapID:     row.YapID,
		UserID:    row.UserID,
		Content:   row.Content,
		Media:     row.Media,
		Hashtags:  row.Hashtags,
		Mentions:  row.Mentions,
		Longitude: row.Longitude,
		Latitude:  row.Latitude,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func ConvertGetListYapsByUserRow(row repository.ListYapsByUserRow) Yap {
	return Yap{
		YapID:     row.YapID,
		UserID:    row.UserID,
		Content:   row.Content,
		Media:     row.Media,
		Hashtags:  row.Hashtags,
		Mentions:  row.Mentions,
		Longitude: row.Longitude,
		Latitude:  row.Latitude,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func ConvertUpdateYapRow(row repository.UpdateYapRow) Yap {
	return Yap{
		YapID:     row.YapID,
		UserID:    row.UserID,
		Content:   row.Content,
		Media:     row.Media,
		Hashtags:  row.Hashtags,
		Mentions:  row.Mentions,
		Longitude: row.Longitude,
		Latitude:  row.Latitude,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
