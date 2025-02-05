package users

import (
	"context"
	"errors"
	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"net/http"
	"time"
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

func (h *UserHandler) UpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	var input UpdateEmailRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	user, err := h.service.UpdateEmail(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case errors.Is(err, ErrContextNotFound):
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		case errors.Is(err, ErrDuplicateEmail):
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

// ForgotPassword is a placeholder for handling password resets for logged-out users.
// 🚨 This function is still a work in progress (WIP) and currently lacks security measures,
// such as email verification or token validation.
// As a result, it will not be exposed as an endpoint until proper security is implemented.
func (h *UserHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input ForgotPasswordRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	err = h.service.ForgotPassword(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "password changed successfully. " +
		"plese log in with your new credentials."},
		nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input ResetPasswordRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	err = h.service.ResetPassword(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case errors.Is(err, ErrContextNotFound):
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		case errors.Is(err, ErrInvalidPassword):
			apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	// clear context after password reset
	// TODO : needs a way to invalidate the jwt as well
	h.clearAuthContext(w, r)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "password changed successfully. " +
		"please log in with your new credentials."}, nil)
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
		switch {
		case errors.Is(err, ErrUserNotFound):
			apierror.GlobalErrorHandler.NotFoundResponse(w, r)
		case errors.Is(err, ErrContextNotFound):
			apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
		case errors.Is(err, ErrInvalidPassword):
			apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
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
