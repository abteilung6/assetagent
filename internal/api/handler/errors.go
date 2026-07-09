package handler

import (
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/service"
)

func writeValidationError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, gen.Error{
		Error:   "validation_failed",
		Message: message,
	})
}

func writeInternalError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, gen.Error{
		Error:   "internal_error",
		Message: message,
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if mapServiceError(w, err) {
		return
	}
	writeInternalError(w, "failed to list transactions")
}

func writeChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chat.ErrEmptyMessages),
		errors.Is(err, chat.ErrInvalidRole):
		writeValidationError(w, err.Error())
	default:
		writeInternalError(w, "failed to process chat request")
	}
}

func mapServiceError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, service.ErrInvalidLimit),
		errors.Is(err, service.ErrInvalidOffset),
		errors.Is(err, service.ErrInvalidSort),
		errors.Is(err, service.ErrInvalidAmountRange),
		errors.Is(err, service.ErrInvalidMinAmount),
		errors.Is(err, service.ErrInvalidMaxAmount):
		writeValidationError(w, err.Error())
		return true
	default:
		return false
	}
}

func APIErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *gen.InvalidParamFormatError
	if errors.As(err, &invalid) {
		writeValidationError(w, invalid.Error())
		return
	}

	writeValidationError(w, err.Error())
}
