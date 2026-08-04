package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/repository"
)

const (
	monthCashflowToolName = "get_month_cashflow"
	categorySpendToolName = "get_category_spend"
	oneOffImpactToolName  = "get_one_off_impact"
)

var yyyyMMPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}$`)

type BaselineInsights interface {
	CategorySpend(ctx context.Context, from, to time.Time, limit int) ([]repository.CategorySpendPoint, error)
	OneOffImpact(ctx context.Context, from, to time.Time) (repository.OneOffExpenseImpact, error)
}

type monthCashflowArgs struct {
	YYYYMM string `json:"yyyy_mm"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type monthCashflowResult struct {
	OK                bool     `json:"ok"`
	Income            string   `json:"income"`
	Expenses          string   `json:"expenses"`
	Net               string   `json:"net"`
	Currency          string   `json:"currency"`
	Period            period   `json:"period"`
	YYYYMM            string   `json:"yyyy_mm,omitempty"`
	AccountsIncluded  []string `json:"accounts_included"`
	TransfersExcluded bool     `json:"transfers_excluded"`
	Calculation       string   `json:"calculation"`
	Confidence        string   `json:"confidence"`
	EvidenceIDs       []string `json:"evidence_ids"`
	DeepLink          string   `json:"deep_link,omitempty"`
}

type categorySpendArgs struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Limit *int   `json:"limit"`
}

type categorySpendResult struct {
	OK       bool                  `json:"ok"`
	Period   period                `json:"period"`
	Currency string                `json:"currency"`
	Items    []categorySpendItem   `json:"items"`
	DeepLink string                `json:"deep_link,omitempty"`
}

type categorySpendItem struct {
	CategorySlug     string `json:"category_slug"`
	CategoryName     string `json:"category_name"`
	Total            string `json:"total"`
	TransactionCount int64  `json:"transaction_count"`
}

type oneOffImpactArgs struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type oneOffImpactResult struct {
	OK           bool   `json:"ok"`
	Period       period `json:"period"`
	Count        int64  `json:"count"`
	ExpenseTotal string `json:"expense_total"`
	Currency     string `json:"currency"`
	DeepLink     string `json:"deep_link,omitempty"`
}

func monthCashflowTool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: monthCashflowToolName,
			Description: "Get transfer-aware income, expenses, and net for one calendar month (yyyy_mm) or an explicit from/to range. Prefer for month story / 'what kind of month' questions.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"yyyy_mm": {"type": "string", "description": "Calendar month YYYY-MM. Prefer when the user names a month."},
					"from": {"type": "string", "description": "Start date YYYY-MM-DD inclusive (alternative to yyyy_mm)."},
					"to": {"type": "string", "description": "End date YYYY-MM-DD inclusive (alternative to yyyy_mm)."}
				}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runMonthCashflow(ctx, reports, raw)
		},
	}
}

func categorySpendTool(insights BaselineInsights) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: categorySpendToolName,
			Description: "Top expense categories for a booking-date range. Prefer for 'where did money go' questions about a month or period.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date YYYY-MM-DD inclusive."},
					"to": {"type": "string", "description": "End date YYYY-MM-DD inclusive."},
					"limit": {"type": "integer", "description": "Max categories (default 8, max 20)."}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runCategorySpend(ctx, insights, raw)
		},
	}
}

func oneOffImpactTool(insights BaselineInsights) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: oneOffImpactToolName,
			Description: "Count and total of one-off (non-recurring) expenses in a booking period. Prefer when explaining unusual months versus Baseline.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date YYYY-MM-DD inclusive."},
					"to": {"type": "string", "description": "End date YYYY-MM-DD inclusive."}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runOneOffImpact(ctx, insights, raw)
		},
	}
}

func runMonthCashflow(ctx context.Context, reports Reports, raw json.RawMessage) (monthCashflowResult, error) {
	if reports == nil {
		return monthCashflowResult{}, errors.New("reports repository is not configured")
	}

	var args monthCashflowArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return monthCashflowResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, yyyyMM, err := parseMonthOrRange(args.YYYYMM, args.From, args.To)
	if err != nil {
		return monthCashflowResult{}, err
	}

	evidence, err := reports.GetCashflowV2Evidence(ctx, from, to)
	if err != nil {
		return monthCashflowResult{}, err
	}

	accounts := evidence.AccountsIncluded
	if accounts == nil {
		accounts = []string{}
	}
	ids := evidence.EvidenceIDs
	if ids == nil {
		ids = []string{}
	}

	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")
	deepLink := ""
	if yyyyMM != "" {
		deepLink = "/baseline/months/" + yyyyMM
	}

	return monthCashflowResult{
		OK:       true,
		Income:   decimalString(evidence.Income),
		Expenses: decimalString(evidence.Expenses),
		Net:      decimalString(evidence.Net),
		Currency: evidence.Currency,
		Period: period{
			From: fromStr,
			To:   toStr,
		},
		YYYYMM:            yyyyMM,
		AccountsIncluded:  accounts,
		TransfersExcluded: evidence.TransfersExcluded,
		Calculation:       evidence.Calculation,
		Confidence:        evidence.Confidence,
		EvidenceIDs:       ids,
		DeepLink:          deepLink,
	}, nil
}

func runCategorySpend(ctx context.Context, insights BaselineInsights, raw json.RawMessage) (categorySpendResult, error) {
	if insights == nil {
		return categorySpendResult{}, errors.New("baseline insights are not configured")
	}

	var args categorySpendArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return categorySpendResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return categorySpendResult{}, err
	}

	limit := 8
	if args.Limit != nil {
		limit = *args.Limit
	}

	items, err := insights.CategorySpend(ctx, from, to, limit)
	if err != nil {
		return categorySpendResult{}, err
	}

	out := make([]categorySpendItem, 0, len(items))
	for _, item := range items {
		out = append(out, categorySpendItem{
			CategorySlug:     item.CategorySlug,
			CategoryName:     item.CategoryName,
			Total:            decimalString(item.Total),
			TransactionCount: item.TransactionCount,
		})
	}

	yyyyMM := yyyyMMIfCalendarMonth(from, to)
	deepLink := ""
	if yyyyMM != "" {
		deepLink = "/baseline/months/" + yyyyMM
	}

	return categorySpendResult{
		OK: true,
		Period: period{
			From: args.From,
			To:   args.To,
		},
		Currency: "EUR",
		Items:    out,
		DeepLink: deepLink,
	}, nil
}

func runOneOffImpact(ctx context.Context, insights BaselineInsights, raw json.RawMessage) (oneOffImpactResult, error) {
	if insights == nil {
		return oneOffImpactResult{}, errors.New("baseline insights are not configured")
	}

	var args oneOffImpactArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return oneOffImpactResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return oneOffImpactResult{}, err
	}

	impact, err := insights.OneOffImpact(ctx, from, to)
	if err != nil {
		return oneOffImpactResult{}, err
	}

	yyyyMM := yyyyMMIfCalendarMonth(from, to)
	deepLink := ""
	if yyyyMM != "" {
		deepLink = "/baseline/months/" + yyyyMM
	}

	return oneOffImpactResult{
		OK: true,
		Period: period{
			From: args.From,
			To:   args.To,
		},
		Count:        impact.Count,
		ExpenseTotal: decimalString(impact.ExpenseTotal),
		Currency:     "EUR",
		DeepLink:     deepLink,
	}, nil
}

func parseMonthOrRange(yyyyMM, fromRaw, toRaw string) (time.Time, time.Time, string, error) {
	if yyyyMM != "" {
		from, to, err := calendarMonthBounds(yyyyMM)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
		return from, to, yyyyMM, nil
	}
	from, to, err := parseDateRange(fromRaw, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	return from, to, yyyyMMIfCalendarMonth(from, to), nil
}

func calendarMonthBounds(yyyyMM string) (time.Time, time.Time, error) {
	if !yyyyMMPattern.MatchString(yyyyMM) {
		return time.Time{}, time.Time{}, errors.New("yyyy_mm must be YYYY-MM")
	}
	from, err := time.Parse("2006-01-02", yyyyMM+"-01")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid yyyy_mm: %w", err)
	}
	to := from.AddDate(0, 1, -1)
	return from, to, nil
}

func yyyyMMIfCalendarMonth(from, to time.Time) string {
	if from.Day() != 1 {
		return ""
	}
	last := from.AddDate(0, 1, -1)
	if to.Year() != last.Year() || to.Month() != last.Month() || to.Day() != last.Day() {
		return ""
	}
	return from.Format("2006-01")
}
