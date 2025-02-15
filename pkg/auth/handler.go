package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthAPI struct {
	dbpool *pgxpool.Pool
	rdb    *redis.Client
}

func New(dbpool *pgxpool.Pool, rdb *redis.Client) *AuthAPI {
	return &AuthAPI{
		dbpool,
		rdb,
	}
}

func (api *AuthAPI) RegisterUser(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(time.Hour * 24 * 7)
	var input AuthInput
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()

	if input.validate(true, v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	jwt, err := register(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	cookie := http.Cookie{
		Name:     "jwt",
		Value:    jwt,
		Expires:  exp,
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"jwt": jwt}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (api *AuthAPI) LoginUser(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(time.Hour * 24 * 7)
	var input AuthInput
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()

	if input.validate(false, v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	jwt, err := login(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows), errors.Is(err, ErrInvalidCredentials):
			apierror.GlobalErrorHandler.InvalidCredentialsResponse(w, r)
		default:
			apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		}
		return
	}

	cookie := http.Cookie{
		Name:     "jwt",
		Value:    jwt,
		Expires:  exp,
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"jwt": jwt}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (api *AuthAPI) LogoutUser(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		apierror.GlobalErrorHandler.InvalidCredentialsResponse(w, r)
		return
	}

	err := api.rdb.Set(r.Context(), fmt.Sprintf("jwt:blacklist:%s", authHeader), authHeader, 7*24*time.Hour).Err()
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}

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

// ForgotPassword is a placeholder for handling password resets for logged-out users.
// 🚨 This function is still a work in progress (WIP) and currently lacks security measures,
// such as email verification or token validation.
// As a result, it will not be exposed as an endpoint until proper security is implemented.
func (api *AuthAPI) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input ForgotPasswordInput
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateForgotPassword(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err := forgorPassword(r.Context(), api.dbpool, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "password updated successfully"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}
