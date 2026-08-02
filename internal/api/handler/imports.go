package handler

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const maxImportUploadBytes = 32 << 20 // 32 MiB

func (h *Handler) PostImportsPreview(w http.ResponseWriter, r *http.Request) {
	data, filename, ok := readImportUpload(w, r)
	if !ok {
		return
	}

	preview, err := service.PreviewBytes(data, filename)
	if err != nil {
		writeImportPreviewError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIImportPreview(preview))
}

func (h *Handler) PostImports(w http.ResponseWriter, r *http.Request) {
	if h.importer == nil {
		writeInternalError(w, "import service is not configured")
		return
	}

	data, filename, ok := readImportUpload(w, r)
	if !ok {
		return
	}

	opts := domain.ImportOptions{
		AccountName: strings.TrimSpace(r.FormValue("account_name")),
		PreviewHash: strings.TrimSpace(r.FormValue("preview_hash")),
	}
	if rawID := strings.TrimSpace(r.FormValue("account_id")); rawID != "" {
		accountID, err := uuid.Parse(rawID)
		if err != nil {
			writeValidationError(w, "account_id must be a valid UUID")
			return
		}
		opts.AccountID = accountID
	}

	result, err := h.importer.ImportBytes(r.Context(), data, filename, opts)
	if err != nil {
		writeImportCommitError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIImportCommit(result))
}

func readImportUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	if err := r.ParseMultipartForm(maxImportUploadBytes); err != nil {
		writeValidationError(w, "invalid multipart form")
		return nil, "", false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidationError(w, "file is required")
		return nil, "", false
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxImportUploadBytes+1))
	if err != nil {
		writeInternalError(w, "failed to read upload")
		return nil, "", false
	}
	if len(data) == 0 {
		writeValidationError(w, "file is empty")
		return nil, "", false
	}
	if len(data) > maxImportUploadBytes {
		writeValidationError(w, "file exceeds maximum size")
		return nil, "", false
	}

	filename := "upload.csv"
	if header != nil && header.Filename != "" {
		filename = filepath.Base(header.Filename)
	}
	return data, filename, true
}

func writeImportPreviewError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "empty file"),
		strings.Contains(msg, "parse csv"),
		strings.Contains(msg, "csv:"):
		writeValidationError(w, msg)
	default:
		writeInternalError(w, "failed to preview import")
	}
}

func writeImportCommitError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrPreviewHashMismatch),
		errors.Is(err, service.ErrAccountNotFound):
		writeValidationError(w, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "empty file") ||
			strings.Contains(msg, "parse csv") ||
			strings.Contains(msg, "csv:") {
			writeValidationError(w, msg)
			return
		}
		writeInternalError(w, "failed to commit import")
	}
}

func toAPIImportCommit(result domain.ImportResult) gen.ImportCommitResponse {
	return gen.ImportCommitResponse{
		ImportRunId: result.ImportRunID,
		AccountId:   result.AccountID,
		AccountName: result.AccountName,
		Rows:        result.Rows,
		Inserted:    result.Inserted,
		Duplicates:  result.Duplicates,
	}
}

func toAPIImportPreview(preview domain.ImportPreview) gen.ImportPreviewResponse {
	samples := make([]gen.ImportPreviewSampleRow, len(preview.SampleRows))
	for i, row := range preview.SampleRows {
		samples[i] = gen.ImportPreviewSampleRow{
			Line:         row.Line,
			BookingDate:  openapi_types.Date{Time: row.BookingDate},
			Counterparty: row.Counterparty,
			Purpose:      row.Purpose,
			Amount:       row.Amount,
			Currency:     row.Currency,
		}
	}

	invalid := make([]gen.ImportPreviewInvalidRow, len(preview.InvalidRows))
	for i, row := range preview.InvalidRows {
		invalid[i] = gen.ImportPreviewInvalidRow{
			Line:    row.Line,
			Message: row.Message,
		}
		if row.Field != "" {
			field := row.Field
			invalid[i].Field = &field
		}
	}

	warnings := preview.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	resp := gen.ImportPreviewResponse{
		FileHash:         preview.FileHash,
		SourceFilename:   preview.SourceFilename,
		ParserName:       preview.ParserName,
		ParserVersion:    preview.ParserVersion,
		SuggestedAccount: preview.SuggestedAccount,
		RowTotal:         preview.RowTotal,
		RowValid:         preview.RowValid,
		RowInvalid:       preview.RowInvalid,
		SampleRows:       samples,
		InvalidRows:      invalid,
		Warnings:         warnings,
	}

	if preview.PeriodFrom != nil && preview.PeriodTo != nil {
		resp.Period = &gen.ImportPreviewPeriod{
			From: openapi_types.Date{Time: *preview.PeriodFrom},
			To:   openapi_types.Date{Time: *preview.PeriodTo},
		}
	}

	return resp
}
