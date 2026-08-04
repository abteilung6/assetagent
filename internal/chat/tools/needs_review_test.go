package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/domain"
)

type fakeTransfers struct {
	items []domain.TransferCandidate
}

func (f *fakeTransfers) ListCandidates(ctx context.Context) ([]domain.TransferCandidate, error) {
	return f.items, nil
}

type fakeQueue struct {
	items []domain.ClassificationQueueItem
}

func (f *fakeQueue) ListQueue(ctx context.Context) ([]domain.ClassificationQueueItem, error) {
	return f.items, nil
}

type fakeUncertain struct {
	items []domain.RecurringSeries
}

func (f *fakeUncertain) Scan(ctx context.Context) (domain.RecurringScanResult, error) {
	return domain.RecurringScanResult{}, nil
}

func (f *fakeUncertain) List(ctx context.Context) ([]domain.RecurringSeries, error) {
	return nil, nil
}

func (f *fakeUncertain) ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error) {
	return f.items, nil
}

func TestNeedsReviewSummaryTool(t *testing.T) {
	registry := tools.NewRegistry(tools.Dependencies{
		Reports:   &insightReports{},
		Lister:    &insightLister{},
		Transfers: &fakeTransfers{items: make([]domain.TransferCandidate, 2)},
		Queue:     &fakeQueue{items: make([]domain.ClassificationQueueItem, 5)},
		Recurring: &fakeUncertain{items: make([]domain.RecurringSeries, 1)},
	})

	raw, err := registry.Execute(context.Background(), "get_needs_review_summary", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["ok"] != true ||
		result["transfers"] != float64(2) ||
		result["categories"] != float64(5) ||
		result["uncertain_recurring"] != float64(1) ||
		result["total"] != float64(8) {
		t.Fatalf("result = %+v", result)
	}
	if result["deep_link"] != "/review" {
		t.Fatalf("deep_link = %v", result["deep_link"])
	}
}
