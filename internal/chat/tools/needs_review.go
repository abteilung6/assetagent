package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
)

const needsReviewSummaryToolName = "get_needs_review_summary"

type TransferCandidatesSource interface {
	ListCandidates(ctx context.Context) ([]domain.TransferCandidate, error)
}

type UncertainRecurringSource interface {
	ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error)
}

type ClassificationQueueSource interface {
	ListQueue(ctx context.Context) ([]domain.ClassificationQueueItem, error)
}

type needsReviewSummaryResult struct {
	OK                 bool   `json:"ok"`
	Transfers          int    `json:"transfers"`
	Categories         int    `json:"categories"`
	UncertainRecurring int    `json:"uncertain_recurring"`
	Total              int    `json:"total"`
	DeepLink           string `json:"deep_link"`
	Message            string `json:"message,omitempty"`
}

func needsReviewSummaryTool(
	transfers TransferCandidatesSource,
	classify ClassificationQueueSource,
	recurring UncertainRecurringSource,
) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: needsReviewSummaryToolName,
			Description: "Counts items across Needs review queues (transfers, categories, uncertain recurring). Prefer for overview / 'what should I clear' questions. Read-only — direct the user to /review to act.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runNeedsReviewSummary(ctx, transfers, classify, recurring)
		},
	}
}

func runNeedsReviewSummary(
	ctx context.Context,
	transfers TransferCandidatesSource,
	classify ClassificationQueueSource,
	recurring UncertainRecurringSource,
) (needsReviewSummaryResult, error) {
	if transfers == nil || classify == nil || recurring == nil {
		return needsReviewSummaryResult{}, errors.New("needs review sources are not configured")
	}

	transferItems, err := transfers.ListCandidates(ctx)
	if err != nil {
		return needsReviewSummaryResult{}, err
	}
	categoryItems, err := classify.ListQueue(ctx)
	if err != nil {
		return needsReviewSummaryResult{}, err
	}
	recurringItems, err := recurring.ListUncertain(ctx)
	if err != nil {
		return needsReviewSummaryResult{}, err
	}

	transferCount := len(transferItems)
	categoryCount := len(categoryItems)
	recurringCount := len(recurringItems)
	total := transferCount + categoryCount + recurringCount

	msg := "Open Needs review to confirm or clear items. Chat cannot confirm transfers, categories, or recurring series."
	if total == 0 {
		msg = "Needs review queues are empty."
	}

	return needsReviewSummaryResult{
		OK:                 true,
		Transfers:          transferCount,
		Categories:         categoryCount,
		UncertainRecurring: recurringCount,
		Total:              total,
		DeepLink:           "/review",
		Message:            msg,
	}, nil
}
