package yap

import (
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// TODO : add retrieve user from context functionality
func (h *Handler) CreateYap(w http.ResponseWriter, r *http.Request) {
	//userID := r.Context().Value("user_id").(pgtype.UUID)

	var input CreateYapRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	ValidateYapContent(v, input.Content)

	if !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := h.service.CreateYap(r.Context(), input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
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
		switch err {
		case ErrYapNotFound:
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *Handler) ListsYapByUser(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	userIDstr := qs.Get("user_id")

	var userID pgtype.UUID

	if userIDstr != "" {
		err := userID.Scan(userIDstr)
		if err != nil {
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
			return
		}
		// TODO : check if user exists
	} else {
		// TODO : get user from context
	}

	var input ListYapsRequest

	v := util.NewValidator()
	input.Limit = int32(util.ReadInt(qs, "limit", 20, v))
	input.Offset = int32(util.ReadInt(qs, "offset", 0, v))
	if !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yaps, err := h.service.ListYapsByUser(r.Context(), userID, input)
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

func (h *Handler) UpdateYap(w http.ResponseWriter, r *http.Request) {
	yapID, err := util.ParseUUIDParam(r, "/api/v1/yaps/")
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	var input UpdateYapRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	input.YapID = yapID

	v := util.NewValidator()
	ValidateYapContent(v, input.Content)
	if !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	yap, err := h.service.UpdateYap(r.Context(), input)
	if err != nil {
		switch err {
		case ErrYapNotFound:
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case ErrUnauthorizedYapper:
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *Handler) DeleteYap(w http.ResponseWriter, r *http.Request) {
	var input DeleteYapRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	err := h.service.DeleteYap(r.Context(), input.YapID, input.UserID)
	if err != nil {
		switch err {
		case ErrYapNotFound:
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case ErrUnauthorizedYapper:
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "yap successfully unyapped"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	}
}
