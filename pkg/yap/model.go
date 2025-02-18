package yap

import (
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"strings"
)

type CreateYapRequest struct {
	Content string `json:"content" validate:"required,max=140"`
}

type YapResponse struct {
	YapID     pgtype.UUID        `json:"yap_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Content   string             `json:"content"`
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

	return v.Errors
}

func (input *UpdateYapRequest) validateYapContent(v *util.Validator) map[string]string {
	v.Check(len(input.Content) > 0, "content", "must be greater than zero")
	v.Check(len(input.Content) <= 140, "content", "must not be greater than 140")
	v.Check(len(strings.TrimSpace(input.Content)) > 0, "content", "must not be blank")
	v.Check(&input.YapID != nil, "yap_id", "must provide yap_id")

	return v.Errors
}
