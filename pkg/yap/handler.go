package yap

import (
	"errors"
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/chocological13/yapper-backend/pkg/util"
	"mime/multipart"
	"net/http"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
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

	err = util.WriteJSON(w, http.StatusCreated, util.Envelope{"media": mediaDetails}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) CreateYap(w http.ResponseWriter, r *http.Request) {
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
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusCreated, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *Handler) GetYapByID(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	yap, err := h.service.GetYapByID(r.Context(), yapID)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *Handler) ListYapsByUser(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	userIDstr := qs.Get("user_id")

	if userIDstr == "" {
		apierror.GlobalErrorHandler.WriteError(w, r, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	h.fetchYapsByUser(w, r, userIDstr)
}

func (h *Handler) ListMyYaps(w http.ResponseWriter, r *http.Request) {
	h.fetchYapsByUser(w, r, "")
}

func (h *Handler) UpdateYap(w http.ResponseWriter, r *http.Request) {
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
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *Handler) DeleteYap(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	err = h.service.DeleteYap(r.Context(), yapID)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "yap successfully unyapped"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	}
}

// -------------------- 🔽 HELPER FUNCTIONS BELOW 🔽 --------------------

// handleErrors handles.. errors
func handleErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrYapNotFound):
		apierror.GlobalErrorHandler.NotFoundResponse(w, r)
	case errors.Is(err, ErrUnauthorizedYapper):
		apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, media.ErrMediaNotFound):
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	default:
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

// parseMultipartForm extracts data from a multipart request
func parseMultipartForm(r *http.Request) (*multipart.FileHeader, error) {
	// Extract media file
	file, header, err := r.FormFile("media")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return header, nil
}

// fetchYapsByUser fetches a list of yaps that have been yapped by a specified user
func (h *Handler) fetchYapsByUser(w http.ResponseWriter, r *http.Request, userIDstr string) {
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
