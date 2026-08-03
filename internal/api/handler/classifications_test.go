package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGetClassificationQueue_ok(t *testing.T) {
	txID := uuid.New()
	router := newClassifyTestRouter(&stubClassifyService{
		queue: []domain.ClassificationQueueItem{{
			TransactionID: txID,
			BookingDate:   time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			Amount:        decimal.RequireFromString("-150.00"),
			Counterparty:  "Unknown Shop",
			CategorySlug:  "other",
			CategoryName:  "Sonstiges",
			Source:        domain.ClassificationSourceUnresolved,
			Confidence:    domain.ClassificationConfidenceLow,
		}},
	}, &stubCategoryService{categories: []domain.Category{{
		ID: uuid.New(), Slug: "other", DisplayName: "Sonstiges", Kind: "other", IsSystem: true,
	}}})

	req := httptest.NewRequest(http.MethodGet, "/api/classifications/queue", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp gen.ClassificationQueueListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].TransactionId != txID {
		t.Fatalf("data = %+v", resp.Data)
	}
}

func TestPostClassificationCorrect_ok(t *testing.T) {
	txID := uuid.New()
	router := newClassifyTestRouter(&stubClassifyService{
		correct: domain.ClassifyCorrectResult{
			TransactionID: txID,
			CategorySlug:  "groceries",
			RuleCreated:   true,
		},
	}, nil)

	body := `{"category_slug":"groceries","apply_to_merchant":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/classifications/"+txID.String()+"/correct", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp gen.ClassificationCorrectResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.RuleCreated || resp.CategorySlug != "groceries" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestGetCategories_ok(t *testing.T) {
	router := newClassifyTestRouter(nil, &stubCategoryService{categories: []domain.Category{{
		ID: uuid.New(), Slug: "income", DisplayName: "Einkommen", Kind: "income", IsSystem: true,
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp gen.CategoryListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Slug != "income" {
		t.Fatalf("data = %+v", resp.Data)
	}
}

func newClassifyTestRouter(classify handler.ClassifyService, categories handler.CategoryService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, nil, classify, categories, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})
	return router
}

type stubClassifyService struct {
	err     error
	queue   []domain.ClassificationQueueItem
	correct domain.ClassifyCorrectResult
}

func (s *stubClassifyService) ListQueue(ctx context.Context) ([]domain.ClassificationQueueItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.queue, nil
}

func (s *stubClassifyService) Correct(ctx context.Context, txID uuid.UUID, opts domain.ClassifyCorrectOptions) (domain.ClassifyCorrectResult, error) {
	if s.err != nil {
		return domain.ClassifyCorrectResult{}, s.err
	}
	return s.correct, nil
}

type stubCategoryService struct {
	categories []domain.Category
}

func (s *stubCategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.categories, nil
}
