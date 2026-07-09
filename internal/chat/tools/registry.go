package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
)

var ErrUnknownTool = errors.New("unknown tool")

type Reports interface {
	GetCashflow(ctx context.Context, from, to time.Time) (domain.CashflowReport, error)
	GetTopCounterparties(ctx context.Context, from, to time.Time, limit int) ([]domain.CounterpartySpend, error)
}

type TransactionLister interface {
	List(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
}

type Dependencies struct {
	Reports Reports
	Lister  TransactionLister
}

type Registry struct {
	tools map[string]toolEntry
}

type toolEntry struct {
	definition llm.Tool
	handler    func(context.Context, json.RawMessage) (any, error)
}

func NewRegistry(deps Dependencies) *Registry {
	r := &Registry{tools: make(map[string]toolEntry)}
	r.register(cashflowTool(deps.Reports))
	r.register(counterpartiesTool(deps.Reports))
	r.register(searchTool(deps.Lister))
	return r
}

func (r *Registry) Tools() []llm.Tool {
	out := make([]llm.Tool, 0, len(r.tools))
	for _, entry := range r.tools {
		out = append(out, entry.definition)
	}
	return out
}

func (r *Registry) Execute(
	ctx context.Context,
	name string,
	args json.RawMessage,
) (json.RawMessage, error) {
	entry, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownTool, name)
	}

	result, err := entry.handler(ctx, args)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode tool result: %w", err)
	}

	return encoded, nil
}

func (r *Registry) register(entry toolEntry) {
	r.tools[entry.definition.Name] = entry
}

func parseDateRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, errors.New("from and to are required (YYYY-MM-DD)")
	}

	from, err := time.Parse("2006-01-02", fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %w", err)
	}
	to, err := time.Parse("2006-01-02", toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %w", err)
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, errors.New("to must be on or after from")
	}

	return from, to, nil
}

func decimalString(value interface{ String() string }) string {
	return value.String()
}
