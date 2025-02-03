package yap

import (
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	"strings"
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
		apierror.Write(w, http.StatusBadRequest, err.Error())
		return
	}

	v := util.NewValidator()
	ValidateYapContent(v, input.Content)

	if !v.Valid() {
		apierror.FailedValidationResponse(w, v.Errors)
		return
	}

	yap, err := h.service.CreateYap(r.Context(), input)
	if err != nil {
		apierror.ServerErrorResponse(w)
		return
	}

	err = util.WriteJSON(w, http.StatusCreated, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.ServerErrorResponse(w)
		return
	}
}

func (h *Handler) GetYapByID(w http.ResponseWriter, r *http.Request) {
	// TODO : use a helper to parse URL params later
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/yaps/")

	uuidStr := strings.Split(path, "/")[0]

	var yapID pgtype.UUID
	err := yapID.Scan(uuidStr)
	if err != nil {
		apierror.ServerErrorResponse(w)
		return
	}

	yap, err := h.service.GetYapByID(r.Context(), yapID)
	if err != nil {
		switch err {
		case ErrYapNotFound:
			apierror.NotFoundResponse(w)
		default:
			apierror.ServerErrorResponse(w)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"yap": yap}, nil)
	if err != nil {
		apierror.ServerErrorResponse(w)
		return
	}
}
