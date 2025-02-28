package yap

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type YapHandler struct {
	dbpool       *pgxpool.Pool
	errorHandler *apierror.ErrorHandler
	mediaService media.MediaService
}

func NewYapHandler(dbpool *pgxpool.Pool, errorHandler *apierror.ErrorHandler, mediaService media.MediaService) *YapHandler {
	return &YapHandler{
		dbpool,
		errorHandler,
		mediaService,
	}
}

func (h *YapHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	var userID pgtype.UUID
	if err := userID.Scan(r.Context().Value("sub")); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	file, err := parseMultipartForm(r)
	if err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if validateFile(v, file); !v.Valid() {
		h.errorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	mediaDetails, err := UploadMedia(r.Context(), h.mediaService, file, userID)
	if err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	h.respondJSON(w, r, http.StatusCreated, util.Envelope{"media": mediaDetails})
}

func (h *YapHandler) CreateYap(w http.ResponseWriter, r *http.Request) {
	var userID pgtype.UUID
	if err := userID.Scan(r.Context().Value("sub")); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	var input CreateYapRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateYapContent(v); !v.Valid() {
		h.errorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := CreateYap(r.Context(), h.dbpool, h.mediaService, input, userID)
	if err != nil {
		h.handleServiceErrors(w, r, err)
		return
	}

	h.respondJSON(w, r, http.StatusCreated, util.Envelope{"yap": yap})
}

func (h *YapHandler) GetYapByID(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/yaps/")
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}

	yap, err := GetYapByID(r.Context(), h.dbpool, h.mediaService, yapID)

	if err != nil {
		h.handleServiceErrors(w, r, err)
		return
	}

	h.respondJSON(w, r, http.StatusOK, util.Envelope{"yap": yap})
}

func (h *YapHandler) ListYapsByUser(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	userIDstr := qs.Get("user_id")

	if userIDstr == "" {
		h.errorHandler.BadRequestResponse(w, r, errors.New("user_id query parameter is required"))
		return
	}

	h.fetchYapsByUser(w, r, userIDstr)
}

func (h *YapHandler) UpdateYap(w http.ResponseWriter, r *http.Request) {
	var userID pgtype.UUID
	if err := userID.Scan(r.Context().Value("sub")); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	yapID, err := util.ParseUUIDParam(r, "/yaps/")
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}

	var input UpdateYapRequest
	if err = util.ReadJSON(w, r, &input); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateYapContent(v); !v.Valid() {
		h.errorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := UpdateYap(r.Context(), h.dbpool, h.mediaService, yapID, input, userID)
	if err != nil {
		h.handleServiceErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *YapHandler) DeleteYap(w http.ResponseWriter, r *http.Request) {
	var userID pgtype.UUID
	if err := userID.Scan(r.Context().Value("sub")); err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	yapID, err := util.ParseUUIDParam(r, "/yaps/")
	if err != nil {
		h.errorHandler.BadRequestResponse(w, r, err)
		return
	}

	err = DeleteYap(r.Context(), h.dbpool, h.mediaService, yapID, userID)
	if err != nil {
		h.handleServiceErrors(w, r, err)
		return
	}

	h.respondJSON(w, r, http.StatusOK, util.Envelope{"message": "yap successfully unyapped"})
}

// -------------------- 🔽 HELPER FUNCTIONS BELOW 🔽 --------------------

// parseMultipartForm extracts data from a multipart request
func parseMultipartForm(r *http.Request) (*multipart.FileHeader, error) {
	file, header, err := r.FormFile("media")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return header, nil
}

// fetchYapsByUser fetches a list of yaps that have been yapped by a specified user
func (h *YapHandler) fetchYapsByUser(w http.ResponseWriter, r *http.Request, userIDstr string) {
	qs := r.URL.Query()

	var input ListYapsRequest

	input.UserID = userIDstr
	println(input.UserID)

	v := util.NewValidator()
	input.Limit = int32(util.ReadInt(qs, "limit", 20, v))
	input.Offset = int32(util.ReadInt(qs, "offset", 0, v))
	if !v.Valid() {
		h.errorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yaps, err := ListYapsByUser(r.Context(), h.dbpool, h.mediaService, input)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}

	if len(yaps) == 0 {
		h.errorHandler.NotFoundResponse(w, r)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yaps": yaps}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

// handleServiceErrors handles.. errors
func (h *YapHandler) handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrYapNotFound):
		h.errorHandler.NotFoundResponse(w, r)
	case errors.Is(err, ErrUnauthorizedYapper):
		h.errorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, apperrors.ErrInvalidUUID):
		h.errorHandler.BadRequestResponse(w, r, err)
	default:
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *YapHandler) respondJSON(w http.ResponseWriter, r *http.Request, status int, data util.Envelope) {
	if err := util.WriteJSON(w, status, data, nil); err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}
