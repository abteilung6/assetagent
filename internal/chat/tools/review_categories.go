package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/service"
)

const suggestReviewCategoriesToolName = "suggest_review_categories"

type ClassificationSuggester interface {
	SuggestCategories(ctx context.Context) ([]service.CategorySuggestion, error)
}

type suggestReviewCategoriesResult struct {
	OK      bool                              `json:"ok"`
	Count   int                               `json:"count"`
	Applied bool                              `json:"applied"`
	Items   []suggestReviewCategoryItemResult `json:"items"`
	Message string                            `json:"message,omitempty"`
}

type suggestReviewCategoryItemResult struct {
	TransactionID  string `json:"transaction_id"`
	BookingDate    string `json:"booking_date"`
	Amount         string `json:"amount"`
	Counterparty   string `json:"counterparty"`
	Purpose        string `json:"purpose"`
	CurrentSlug    string `json:"current_slug"`
	SuggestedSlug  string `json:"suggested_slug,omitempty"`
	MatchedPattern string `json:"matched_pattern,omitempty"`
	Confidence     string `json:"confidence,omitempty"`
	AutoApplicable bool   `json:"auto_applicable"`
}

func suggestReviewCategoriesTool(source ClassificationSuggester) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: suggestReviewCategoriesToolName,
			Description: "List Needs review category queue items with keyword-based suggestions. Prefer this when helping clear category review. Read-only — tell the user to use Apply suggested categories on Needs review to save.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runSuggestReviewCategories(ctx, source)
		},
	}
}

func runSuggestReviewCategories(
	ctx context.Context,
	source ClassificationSuggester,
) (suggestReviewCategoriesResult, error) {
	if source == nil {
		return suggestReviewCategoriesResult{}, errors.New("classify service is not configured")
	}
	items, err := source.SuggestCategories(ctx)
	if err != nil {
		return suggestReviewCategoriesResult{}, err
	}
	out := make([]suggestReviewCategoryItemResult, 0, len(items))
	for _, item := range items {
		out = append(out, suggestReviewCategoryItemResult{
			TransactionID:  item.TransactionID.String(),
			BookingDate:    item.BookingDate,
			Amount:         item.Amount,
			Counterparty:   item.Counterparty,
			Purpose:        item.Purpose,
			CurrentSlug:    item.CurrentSlug,
			SuggestedSlug:  item.SuggestedSlug,
			MatchedPattern: item.MatchedPattern,
			Confidence:     item.Confidence,
			AutoApplicable: item.AutoApplicable,
		})
	}
	msg := ""
	if len(out) == 0 {
		msg = "Category review queue is empty."
	} else {
		msg = "Suggestions are read-only. Ask the user to open Needs review and click Apply suggested categories to save auto-applicable ones."
	}
	return suggestReviewCategoriesResult{
		OK:      true,
		Count:   len(out),
		Applied: false,
		Items:   out,
		Message: msg,
	}, nil
}
