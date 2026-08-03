package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
)

var ErrUnknownTool = errors.New("unknown tool")

type Reports interface {
	GetCashflow(ctx context.Context, from, to time.Time) (domain.CashflowReport, error)
	GetCashflowV2Evidence(ctx context.Context, from, to time.Time) (domain.CashflowV2Evidence, error)
	GetTopCounterparties(ctx context.Context, from, to time.Time, limit int) ([]domain.CounterpartySpend, error)
}

type TransactionLister interface {
	List(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
}

type Dependencies struct {
	Reports     Reports
	Lister      TransactionLister
	Recurring   recurringSeriesSource
	Baseline    BaselineSource
	MoneyReview MoneyReviewSource
	Forecast    ForecastSource
	Classify    ClassificationSuggester
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
	r.register(cashflowV2Tool(deps.Reports))
	r.register(recurringCostsTool(deps.Recurring, deps.Reports))
	r.register(spendingChangesTool(deps.Reports))
	r.register(anomaliesTool(deps.Recurring, deps.Reports, deps.Lister))
	r.register(counterpartiesTool(deps.Reports))
	r.register(searchTool(deps.Lister))
	r.register(baselineTool(deps.Baseline))
	r.register(moneyReviewTool(deps.MoneyReview))
	r.register(forecastTool(deps.Forecast))
	r.register(suggestReviewCategoriesTool(deps.Classify))
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

func parseOptionalLimit(raw json.RawMessage, key string) (*int, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	field, ok := payload[key]
	if !ok || string(field) == "null" {
		return nil, nil
	}

	return decodeLimitJSON(field)
}

func decodeLimitJSON(data json.RawMessage) (*int, error) {
	var asInt int
	if err := json.Unmarshal(data, &asInt); err == nil {
		return &asInt, nil
	}

	var asFloat float64
	if err := json.Unmarshal(data, &asFloat); err == nil {
		if asFloat != math.Trunc(asFloat) {
			return nil, errors.New("limit must be a whole number")
		}
		limit := int(asFloat)
		return &limit, nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		parsed, err := strconv.Atoi(strings.TrimSpace(asString))
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %q", asString)
		}
		return &parsed, nil
	}

	return nil, errors.New("limit must be a number")
}

func decimalString(value interface{ String() string }) string {
	return value.String()
}
