package yap

import (
	"errors"
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/util"
	"mime/multipart"
	"net/http"
)

type YapHandler struct {
	service YapService
}

func NewYapHandler(service YapService) *YapHandler {
	return &YapHandler{service: service}
}

func (h *YapHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	file, err := parseMultipartForm(r)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if validateFile(v, file); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	mediaDetails, err := h.service.UploadMedia(r.Context(), file)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	respondJSON(w, r, http.StatusCreated, util.Envelope{"media": mediaDetails})
}

func (h *YapHandler) CreateYap(w http.ResponseWriter, r *http.Request) {
	var input CreateYapRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateYapContent(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := h.service.CreateYap(r.Context(), input)
	if err != nil {
		handleServiceErrors(w, r, err)
		return
	}

	respondJSON(w, r, http.StatusCreated, util.Envelope{"yap": yap})
}

func (h *YapHandler) GetYapByID(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	yap, err := h.service.GetYapByID(r.Context(), yapID)
	if err != nil {
		handleServiceErrors(w, r, err)
		return
	}

	respondJSON(w, r, http.StatusOK, util.Envelope{"yap": yap})
}

func (h *YapHandler) ListYapsByUser(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	userIDstr := qs.Get("user_id")

	if userIDstr == "" {
		apierror.GlobalErrorHandler.WriteError(w, r, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	h.fetchYapsByUser(w, r, userIDstr)
}

func (h *YapHandler) ListMyYaps(w http.ResponseWriter, r *http.Request) {
	h.fetchYapsByUser(w, r, "")
}

func (h *YapHandler) UpdateYap(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	var input UpdateYapRequest
	if err = util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateYapContent(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := h.service.UpdateYap(r.Context(), yapID, input)
	if err != nil {
		handleServiceErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *YapHandler) DeleteYap(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	err = h.service.DeleteYap(r.Context(), yapID)
	if err != nil {
		handleServiceErrors(w, r, err)
		return
	}

	respondJSON(w, r, http.StatusOK, util.Envelope{"message": "yap successfully unyapped"})
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

	v := util.NewValidator()
	input.Limit = int32(util.ReadInt(qs, "limit", 20, v))
	input.Offset = int32(util.ReadInt(qs, "offset", 0, v))
	if !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yaps, err := h.service.ListYapsByUser(r.Context(), input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	if len(yaps) == 0 {
		apierror.GlobalErrorHandler.WriteError(w, r, http.StatusNotFound, "this user has not yapped any yap")
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yaps": yaps}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

// handleServiceErrors handles.. errors
func handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrYapNotFound):
		apierror.GlobalErrorHandler.NotFoundResponse(w, r)
	case errors.Is(err, ErrUnauthorizedYapper):
		apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, apperrors.ErrInvalidUUID):
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	default:
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func respondJSON(w http.ResponseWriter, r *http.Request, status int, data util.Envelope) {
	if err := util.WriteJSON(w, status, data, nil); err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}
