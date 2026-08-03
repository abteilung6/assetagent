package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/decisions"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type stubDecisionService struct {
	created service.Decision
	actions []service.Action
}

func (s *stubDecisionService) Create(_ context.Context, req service.CreateDecisionRequest) (service.Decision, error) {
	id := uuid.New()
	actionID := uuid.New()
	s.created = service.Decision{
		ID:        id,
		ReviewID:  req.ReviewID,
		Title:     req.Title,
		DecidedAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		Action: service.Action{
			ID:                   actionID,
			DecisionID:           id,
			Title:                req.Action.Title,
			ExpectedAnnualEffect: req.Action.ExpectedAnnualEffect,
			DueOn:                req.Action.DueOn,
			Status:               decisions.StatusPlanned,
			CreatedAt:            time.Now().UTC(),
			UpdatedAt:            time.Now().UTC(),
		},
	}
	s.actions = []service.Action{s.created.Action}
	return s.created, nil
}

func (s *stubDecisionService) List(context.Context, int) ([]service.Decision, error) {
	if s.created.ID == uuid.Nil {
		return nil, nil
	}
	return []service.Decision{s.created}, nil
}

func (s *stubDecisionService) ListActions(_ context.Context, status *string, _ int) ([]service.Action, error) {
	out := make([]service.Action, 0, len(s.actions))
	for _, a := range s.actions {
		if status != nil && a.Status != *status {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *stubDecisionService) UpdateActionStatus(_ context.Context, id uuid.UUID, req service.UpdateActionStatusRequest) (service.Action, error) {
	for i, a := range s.actions {
		if a.ID != id {
			continue
		}
		if !decisions.CanTransition(a.Status, req.Status) {
			return service.Action{}, service.ErrActionStatusForbidden
		}
		a.Status = req.Status
		s.actions[i] = a
		if s.created.Action.ID == id {
			s.created.Action = a
		}
		return a, nil
	}
	return service.Action{}, service.ErrActionNotFound
}

func TestDecisionCreateAndActionStatus(t *testing.T) {
	t.Parallel()
	svc := &stubDecisionService{}
	router := gen.HandlerWithOptions(
		handler.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, svc),
		gen.ChiServerOptions{},
	)

	reviewID := uuid.New()
	body, _ := json.Marshal(map[string]any{
		"review_id": reviewID.String(),
		"title":     "Cut variable spend",
		"action": map[string]any{
			"title":                  "Reduce groceries by 50€/mo",
			"expected_annual_effect": "600.00",
			"due_on":                 "2026-09-01",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/decisions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/actions?status=planned", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var listed gen.ActionListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data) != 1 {
		t.Fatalf("expected 1 action, got %d", len(listed.Data))
	}

	actionID := listed.Data[0].Id
	statusBody, _ := json.Marshal(map[string]any{"status": "done"})
	statusReq := httptest.NewRequest(http.MethodPost, "/api/actions/"+actionID.String()+"/status", bytes.NewReader(statusBody))
	statusReq.Header.Set("Content-Type", "application/json")
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status update=%d body=%s", statusRec.Code, statusRec.Body.String())
	}
	if svc.actions[0].Status != decisions.StatusDone {
		t.Fatalf("status=%s want done", svc.actions[0].Status)
	}
	if !svc.actions[0].ExpectedAnnualEffect.Equal(decimal.RequireFromString("600.00")) {
		t.Fatalf("effect=%s", svc.actions[0].ExpectedAnnualEffect)
	}
}
