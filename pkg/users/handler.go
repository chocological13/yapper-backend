package users

import (
	"errors"
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"net/http"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Testing purposes only
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	user, err := h.service.GetUser(r.Context(), GetUserRequest{Email: email})
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetCurrentUser(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case errors.Is(err, ErrContextNotFound):
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var input UpdateUserRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	user, err := h.service.UpdateUser(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case errors.Is(err, ErrContextNotFound):
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		case errors.Is(err, ErrDuplicateUsername):
			apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}
