package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGetTransferCandidates_ok(t *testing.T) {
	id := uuid.New()
	router := newTransfersTestRouter(&stubTransferService{
		candidates: []domain.TransferCandidate{
			sampleCandidate(id),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/transfers/candidates", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp gen.TransferCandidateListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Id != id {
		t.Fatalf("data = %+v", resp.Data)
	}
	if resp.Data[0].Amount != "500.00" {
		t.Fatalf("amount = %q", resp.Data[0].Amount)
	}
}

func TestPostTransferConfirm_ok(t *testing.T) {
	id := uuid.New()
	router := newTransfersTestRouter(&stubTransferService{
		pair: domain.TransferPair{
			ID:         id,
			TxOutID:    uuid.New(),
			TxInID:     uuid.New(),
			Status:     domain.TransferStatusConfirmed,
			Confidence: domain.TransferConfidenceExact,
			CreatedAt:  time.Now().UTC(),
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/transfers/"+id.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp gen.TransferPair
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != gen.TransferPairStatusConfirmed {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestPostTransferConfirm_notFound(t *testing.T) {
	router := newTransfersTestRouter(&stubTransferService{err: service.ErrTransferPairNotFound})
	req := httptest.NewRequest(http.MethodPost, "/api/transfers/"+uuid.New().String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPostTransferConfirm_conflict(t *testing.T) {
	router := newTransfersTestRouter(&stubTransferService{err: service.ErrTransferPairNotSuggested})
	req := httptest.NewRequest(http.MethodPost, "/api/transfers/"+uuid.New().String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestPostTransferReject_ok(t *testing.T) {
	id := uuid.New()
	router := newTransfersTestRouter(&stubTransferService{
		pair: domain.TransferPair{
			ID:         id,
			Status:     domain.TransferStatusRejected,
			Confidence: domain.TransferConfidenceProbable,
			CreatedAt:  time.Now().UTC(),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/transfers/"+id.String()+"/reject", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func newTransfersTestRouter(transfers handler.TransferService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, transfers, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})
	return router
}

type stubTransferService struct {
	err        error
	candidates []domain.TransferCandidate
	pair       domain.TransferPair
}

func (s *stubTransferService) ListCandidates(ctx context.Context) ([]domain.TransferCandidate, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.candidates, nil
}

func (s *stubTransferService) Confirm(ctx context.Context, id uuid.UUID) (domain.TransferPair, error) {
	if s.err != nil {
		return domain.TransferPair{}, s.err
	}
	return s.pair, nil
}

func (s *stubTransferService) Reject(ctx context.Context, id uuid.UUID) (domain.TransferPair, error) {
	if s.err != nil {
		return domain.TransferPair{}, s.err
	}
	return s.pair, nil
}

func sampleCandidate(id uuid.UUID) domain.TransferCandidate {
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	return domain.TransferCandidate{
		ID:         id,
		Status:     domain.TransferStatusSuggested,
		Confidence: domain.TransferConfidenceExact,
		Amount:     decimal.RequireFromString("500.00"),
		CreatedAt:  day,
		Out: domain.TransferLegView{
			TransactionID: uuid.New(),
			AccountName:   "Checking",
			BookingDate:   day,
			Amount:        decimal.RequireFromString("-500.00"),
			BookingText:   "UMBUCHUNG",
			Purpose:       "to savings",
		},
		In: domain.TransferLegView{
			TransactionID: uuid.New(),
			AccountName:   "Savings",
			BookingDate:   day,
			Amount:        decimal.RequireFromString("500.00"),
			BookingText:   "UMBUCHUNG",
			Purpose:       "from checking",
		},
	}
}
