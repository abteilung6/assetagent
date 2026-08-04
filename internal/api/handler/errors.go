package handler

import (
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/service"
)

func writeUnauthorizedError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusUnauthorized, gen.Error{
		Error:   "unauthorized",
		Message: message,
	})
}

func writeValidationError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, gen.Error{
		Error:   "validation_failed",
		Message: message,
	})
}

func writeNotFoundError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, gen.Error{
		Error:   "not_found",
		Message: message,
	})
}

func writeConflictError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, gen.Error{
		Error:   "conflict",
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
		errors.Is(err, chat.ErrInvalidRole),
		errors.Is(err, llm.ErrProviderDisabled),
		errors.Is(err, llm.ErrProviderUnknown),
		errors.Is(err, llm.ErrModelNotAllowed),
		errors.Is(err, llm.ErrOpenRouterNoKey):
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
