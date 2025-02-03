package auth

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"

	"github.com/chocological13/yapper-backend/pkg/apierror"
	"github.com/chocological13/yapper-backend/pkg/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthAPI struct {
	dbpool *pgxpool.Pool
}

func New(dbpool *pgxpool.Pool) *AuthAPI {
	return &AuthAPI{
		dbpool,
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

	jwt, err := register(r.Context(), api.dbpool, &input)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
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

	jwt, err := login(r.Context(), api.dbpool, &input)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows), err.Error() == "Invalid credentials":
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
