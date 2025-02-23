package users

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	dbpool       *pgxpool.Pool
	errorHandler *apierror.ErrorHandler
}

func NewUserHandler(dbpool *pgxpool.Pool, errorHandler *apierror.ErrorHandler) *UserHandler {
	return &UserHandler{dbpool, errorHandler}
}

// Testing purposes only
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	user, err := getUser(r.Context(), h.dbpool, GetUserRequest{Email: email})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := getCurrentUser(r.Context(), h.dbpool)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var input UpdateUserRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}

	user, err := updateUser(r.Context(), h.dbpool, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"user": user}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var input DeleteUserRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
		return
	}

	err = deleteUser(r.Context(), h.dbpool, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	// TODO : needs a way to invalidate the jwt as well or a logout function
	h.clearAuthContext(w, r)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "user successfully deleted"}, nil)
	if err != nil {
		h.errorHandler.ServerErrorResponse(w, r, err)
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

func (h *UserHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound):
		h.errorHandler.NotFoundResponse(w, r)
	case errors.Is(err, apperrors.ErrContextNotFound):
		h.errorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, apperrors.ErrDuplicateEmail):
		h.errorHandler.BadRequestResponse(w, r, err)
	default:
		h.errorHandler.ServerErrorResponse(w, r, err)
	}
}
