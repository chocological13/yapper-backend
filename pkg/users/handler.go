package users

import (
	"context"
	"errors"
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/util"
	"net/http"
	"time"
)

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Testing purposes only
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	user, err := h.service.GetUser(r.Context(), GetUserRequest{Email: email})
	if err != nil {
		handleError(w, r, err)
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
		handleError(w, r, err)
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
		handleError(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var input DeleteUserRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	err = h.service.DeleteUser(r.Context(), input)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// TODO : needs a way to invalidate the jwt as well or a logout function
	h.clearAuthContext(w, r)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "user successfully deleted"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

// helper to clear context after reset password
func (h *UserHandler) clearAuthContext(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	*r = *r.WithContext(context.Background())
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound):
		apierror.GlobalErrorHandler.NotFoundResponse(w, r)
	case errors.Is(err, apperrors.ErrContextNotFound):
		apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, apperrors.ErrDuplicateEmail):
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	default:
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}
