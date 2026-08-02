package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestPostImportsPreview_ok(t *testing.T) {
	router := newImportsTestRouter(nil)
	body, contentType := multipartFile(t, "minimal.csv", readFixture(t, "minimal.csv"))

	req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var resp gen.ImportPreviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RowTotal != 6 || resp.RowValid != 6 || resp.RowInvalid != 0 {
		t.Fatalf("counts = total=%d valid=%d invalid=%d", resp.RowTotal, resp.RowValid, resp.RowInvalid)
	}
	if resp.FileHash == "" || resp.SuggestedAccount == "" {
		t.Fatalf("missing hash/account: %+v", resp)
	}
	if resp.Period == nil {
		t.Fatal("expected period")
	}
	if len(resp.SampleRows) == 0 {
		t.Fatal("expected sample rows")
	}
	if resp.SourceFilename != "minimal.csv" {
		t.Fatalf("source_filename = %q", resp.SourceFilename)
	}
}

func TestPostImportsPreview_missingFile(t *testing.T) {
	router := newImportsTestRouter(nil)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("other", "x")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertValidationFailed(t, rec)
}

func TestPostImportsPreview_emptyFile(t *testing.T) {
	router := newImportsTestRouter(nil)
	body, contentType := multipartFile(t, "empty.csv", []byte{})

	req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertValidationFailed(t, rec)
}

func TestPostImportsPreview_invalidCSV(t *testing.T) {
	router := newImportsTestRouter(nil)
	body, contentType := multipartFile(t, "bad.csv", []byte("not-a-csv"))

	req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertValidationFailed(t, rec)
}

func TestPostImports_previewHashMismatch(t *testing.T) {
	importer := &stubImportService{err: service.ErrPreviewHashMismatch}
	router := newImportsTestRouter(importer)
	body, contentType := multipartImport(t, "minimal.csv", readFixture(t, "minimal.csv"), map[string]string{
		"preview_hash": "deadbeef",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/imports", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertValidationFailed(t, rec)
}

func TestPostImports_missingImporter(t *testing.T) {
	router := newImportsTestRouter(nil)
	body, contentType := multipartFile(t, "minimal.csv", readFixture(t, "minimal.csv"))

	req := httptest.NewRequest(http.MethodPost, "/api/imports", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rec.Code, rec.Body.String())
	}
}

func newImportsTestRouter(importer handler.ImportService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, importer), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})
	return router
}

type stubImportService struct {
	err    error
	result domain.ImportResult
}

func (s *stubImportService) ImportBytes(ctx context.Context, data []byte, filename string, opts domain.ImportOptions) (domain.ImportResult, error) {
	if s.err != nil {
		return domain.ImportResult{}, s.err
	}
	return s.result, nil
}

func multipartFile(t *testing.T, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	return multipartImport(t, filename, data, nil)
}

func multipartImport(t *testing.T, filename string, data []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField %s: %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "sparkasse", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}
