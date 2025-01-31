package apierror

import (
	"net/http"

	"github.com/chocological13/yapper-backend/pkg/util"
)

func Write(w http.ResponseWriter, status int, message any) {
	envelope := util.Envelope{"error": message}

	err := util.WriteJSON(w, status, envelope, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ServerErrorResponse(w http.ResponseWriter) {
	msg := "The server encountered a problem and could not process your request"
	Write(w, http.StatusInternalServerError, msg)
}

func NotFoundResponse(w http.ResponseWriter) {
	msg := "The requested resource could not be found"
	Write(w, http.StatusNotFound, msg)
}

func MethodNotAllowedResponse(w http.ResponseWriter) {
	msg := "Unsupported method"
	Write(w, http.StatusMethodNotAllowed, msg)
}

func FailedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	Write(w, http.StatusUnprocessableEntity, errors)
}
