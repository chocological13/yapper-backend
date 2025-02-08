package auth

import (
	"errors"
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
