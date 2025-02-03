package apierror

import (
	"log/slog"
	"net/http"

	"github.com/chocological13/yapper-backend/pkg/util"
)

var GlobalErrorHandler *ErrorHandler // Global variable

// Initialize the global ErrorHandler once
func init() {
	GlobalErrorHandler = &ErrorHandler{logger: slog.Default()}
}

type ErrorHandler struct {
	logger *slog.Logger
}

func (eh *ErrorHandler) logError(r *http.Request, err error) {
	eh.logger.Error("request error",
		"error", err.Error(),
		"method", r.Method,
		"uri", r.RequestURI,
	)
}

func (eh *ErrorHandler) WriteError(w http.ResponseWriter, r *http.Request, status int, message any) {
	envelope := util.Envelope{"error": message}

	err := util.WriteJSON(w, status, envelope, nil)
	if err != nil {
		eh.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (eh *ErrorHandler) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	eh.logError(r, err)
	msg := "The server encountered a problem and could not process your request"
	eh.WriteError(w, r, http.StatusInternalServerError, msg)
}

func (eh *ErrorHandler) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	msg := "The requested resource could not be found"
	eh.WriteError(w, r, http.StatusNotFound, msg)
}

func (eh *ErrorHandler) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Unsupported method"
	eh.WriteError(w, r, http.StatusMethodNotAllowed, msg)
}

func (eh *ErrorHandler) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	eh.WriteError(w, r, http.StatusBadRequest, err.Error())
}

func (eh *ErrorHandler) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Request rate limit exceeded"
	eh.WriteError(w, r, http.StatusTooManyRequests, msg)
}

func (eh *ErrorHandler) FailedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	eh.WriteError(w, r, http.StatusBadRequest, errors)
}

func (eh *ErrorHandler) UnauthorizedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Unauthorized access"
	eh.WriteError(w, r, http.StatusUnauthorized, msg)
}

func (eh *ErrorHandler) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Invalid authentication credentials"
	eh.WriteError(w, r, http.StatusUnauthorized, msg)
}
