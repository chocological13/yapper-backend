package auth

import (
	"net/mail"

	"github.com/chocological13/yapper-backend/pkg/util"
)

type AuthInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

type ResetPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

func (input *AuthInput) validate(isRegistering bool, v *util.Validator) map[string]string {

	if isRegistering {
		v.Check(input.Email != "", "Email", "Must not be empty")
		v.Check(input.Username != "", "Username", "Must not be empty")
		isValidEmail := validateEmail(input.Email)
		v.Check(isValidEmail, "Email", "Email must be valid")
	} else {
		v.Check(input.Email != "" || input.Username != "", "Email or Username", "Must not be empty")
	}

	v.Check(input.Password != "", "Password", "Must not be empty")

	return v.Errors
}

func (input *ForgotPasswordRequest) validateForgotPassword(v *util.Validator) map[string]string {
	v.Check(input.NewPassword != "", "NewPassword", "Must not be empty")
	v.Check(input.Email != "", "Email", "Must not be empty")
	isValidEmail := validateEmail(input.Email)
	v.Check(isValidEmail, "Email", "Email must be valid")

	return v.Errors
}

func (input *ResetPasswordRequest) validateResetPassword(v *util.Validator) map[string]string {
	v.Check(input.NewPassword != "", "NewPassword", "Must not be empty")
	v.Check(input.CurrentPassword != "", "CurrentPassword", "Must not be empty")
	v.Check(input.CurrentPassword != input.NewPassword, "NewPassword, CurrentPassword", "Must not be the same")

	return v.Errors
}

func (input *UpdateEmailRequest) validateUpdateEmail(ctxEmail string, v *util.Validator) map[string]string {
	v.Check(input.NewEmail != "", "NewEmail", "Must not be empty")
	isValidEmail := validateEmail(input.NewEmail)
	v.Check(isValidEmail, "Email", "Email must be valid")
	v.Check(input.Password != "", "Password", "Must not be empty")
	v.Check(ctxEmail != input.NewEmail, "NewEmail", "Email must not be the same as the current email")
	return v.Errors
}

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
