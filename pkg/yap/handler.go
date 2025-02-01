package yap

import (
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
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

	util.WriteJSON(w, http.StatusCreated, util.Envelope{"yap": yap}, nil)
}
