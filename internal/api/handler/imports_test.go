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
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func TestGetImport_notFound(t *testing.T) {
	importer := &stubImportService{err: service.ErrImportRunNotFound}
	router := newImportsTestRouter(importer)

	req := httptest.NewRequest(http.MethodGet, "/api/imports/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.Error
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "not_found" {
		t.Fatalf("error = %q, want not_found", resp.Error)
	}
}

func TestPostImportRollback_conflict(t *testing.T) {
	importer := &stubImportService{err: service.ErrImportRunAlreadyRolledBack}
	router := newImportsTestRouter(importer)

	req := httptest.NewRequest(http.MethodPost, "/api/imports/"+uuid.NewString()+"/rollback", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.Error
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "conflict" {
		t.Fatalf("error = %q, want conflict", resp.Error)
	}
}

func TestGetImports_ok(t *testing.T) {
	runID := uuid.New()
	importer := &stubImportService{
		runs: []domain.ImportRunSummary{{
			ID:             runID,
			AccountID:      uuid.New(),
			SourceFilename: "minimal.csv",
			Status:         domain.ImportRunStatusCommitted,
			RowTotal:       6,
			RowValid:       6,
			RowInserted:    6,
			CreatedAt:      time.Now().UTC(),
		}},
	}
	router := newImportsTestRouter(importer)

	req := httptest.NewRequest(http.MethodGet, "/api/imports?limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.ImportRunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Id != runID {
		t.Fatalf("data = %+v", resp.Data)
	}
}

func newImportsTestRouter(importer handler.ImportService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, importer, nil, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})
	return router
}

type stubImportService struct {
	err    error
	result domain.ImportResult
	runs   []domain.ImportRunSummary
	run    domain.ImportRunSummary
	rb     domain.ImportRollbackResult
}

func (s *stubImportService) ImportBytes(ctx context.Context, data []byte, filename string, opts domain.ImportOptions) (domain.ImportResult, error) {
	if s.err != nil {
		return domain.ImportResult{}, s.err
	}
	return s.result, nil
}

func (s *stubImportService) ListRuns(ctx context.Context, limit int) ([]domain.ImportRunSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.runs, nil
}

func (s *stubImportService) GetRun(ctx context.Context, runID uuid.UUID) (domain.ImportRunSummary, error) {
	if s.err != nil {
		return domain.ImportRunSummary{}, s.err
	}
	return s.run, nil
}

func (s *stubImportService) Rollback(ctx context.Context, runID uuid.UUID) (domain.ImportRollbackResult, error) {
	if s.err != nil {
		return domain.ImportRollbackResult{}, s.err
	}
	return s.rb, nil
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
