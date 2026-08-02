package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const maxImportUploadBytes = 32 << 20 // 32 MiB

func (h *Handler) PostImportsPreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxImportUploadBytes); err != nil {
		writeValidationError(w, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidationError(w, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxImportUploadBytes+1))
	if err != nil {
		writeInternalError(w, "failed to read upload")
		return
	}
	if len(data) == 0 {
		writeValidationError(w, "file is empty")
		return
	}
	if len(data) > maxImportUploadBytes {
		writeValidationError(w, "file exceeds maximum size")
		return
	}

	filename := "upload.csv"
	if header != nil && header.Filename != "" {
		filename = filepath.Base(header.Filename)
	}

	preview, err := service.PreviewBytes(data, filename)
	if err != nil {
		writeImportPreviewError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIImportPreview(preview))
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
