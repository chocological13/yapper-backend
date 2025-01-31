package auth

import (
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
		apierror.Write(w, http.StatusBadRequest, err.Error())
		return
	}

	v := util.NewValidator()
	input.validate(true, v)

	if !v.Valid() {
		http.Error(w, "Invalidddd", http.StatusBadRequest)
		return
	}

	jwt, err := register(r.Context(), api.dbpool, &input)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, err.Error())
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

	util.WriteJSON(w, http.StatusOK, util.Envelope{"jwt": jwt}, nil)
}

func (api *AuthAPI) LoginUser(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(time.Hour * 24 * 7)
	var input AuthInput
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.Write(w, http.StatusBadRequest, err.Error())
		return
	}

	v := util.NewValidator()
	input.validate(false, v)

	if !v.Valid() {
		http.Error(w, "Invalidddd", http.StatusBadRequest)
		return
	}

	jwt, err := login(r.Context(), api.dbpool, &input)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, err.Error())
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

	util.WriteJSON(w, http.StatusOK, util.Envelope{"jwt": jwt}, nil)
}
