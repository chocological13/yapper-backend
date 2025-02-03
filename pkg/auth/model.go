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

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
