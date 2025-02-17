package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"net/http"
	"time"

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
		handleErrors(w, r, err)
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

// InitiateForgotPassword handles the first step of the password reset process for logged-out users.
// It validates the user's request, generates a reset tokens, and (in the future) will send a password reset email.
// 🚨 This function is still a work in progress (WIP) and currently lacks email-sending functionality.
// As a result, it will not be fully functional until the email delivery mechanism is implemented.
func (api *AuthAPI) InitiateForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input ForgotPasswordRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateForgotPasswordRequest(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	tokenString, err := initiateForgorPassword(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"token": tokenString}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (api *AuthAPI) CompleteForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input CompleteForgotPassword
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateForgotPassword(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = completeForgorPassword(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "password updated successfully"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}

}

func (api *AuthAPI) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input ResetPasswordRequest
	err := util.ReadJSON(w, r, &input)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateResetPassword(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = resetPassword(r.Context(), api.dbpool, &input)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	// clear context after password reset
	api.LogoutUser(w, r)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "password changed successfully. " +
		"please log in with your new credentials."}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

// InitiateUpdateUserEmail starts the email update process by generating a verification token.
// 🚨 This function is a work in progress (WIP) and currently does not send the verification email.
// The email sending functionality will be implemented in a future update.
func (api *AuthAPI) InitiateUpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	var input UpdateEmailRequest
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	ctxEmail := r.Context().Value("sub").(string)

	v := util.NewValidator()
	if input.validateUpdateEmail(ctxEmail, v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := initiateUpdateEmail(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		handleErrors(w, r, err)
		return
	}

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"token": token}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

func (api *AuthAPI) CompleteUpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	var input CompleteUpdateEmail
	if err := util.ReadJSON(w, r, &input); err != nil {
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
		return
	}

	v := util.NewValidator()
	if input.validateUpdateEmail(v); !v.Valid() {
		apierror.GlobalErrorHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err := completeUpdateUserEmail(r.Context(), api.dbpool, api.rdb, &input)
	if err != nil {
		handleErrors(w, r, err)
	}

	api.LogoutUser(w, r)

	err = util.WriteJSON(w, http.StatusOK, util.Envelope{"message": "email updated successfully. " +
		"please log back in with your new credentials"}, nil)
	if err != nil {
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}

// helpers
func handleErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound):
		apierror.GlobalErrorHandler.NotFoundResponse(w, r)
	case errors.Is(err, apperrors.ErrContextNotFound):
		apierror.GlobalErrorHandler.UnauthorizedResponse(w, r)
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	case errors.Is(err, ErrInvalidToken):
		apierror.GlobalErrorHandler.BadRequestResponse(w, r, err)
	default:
		apierror.GlobalErrorHandler.ServerErrorResponse(w, r, err)
	}
}
