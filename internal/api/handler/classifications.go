package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type ClassifyService interface {
	ListQueue(ctx context.Context) ([]domain.ClassificationQueueItem, error)
	Correct(ctx context.Context, txID uuid.UUID, opts domain.ClassifyCorrectOptions) (domain.ClassifyCorrectResult, error)
	ApplySuggestions(ctx context.Context) (service.ApplySuggestionsResult, error)
}

type CategoryService interface {
	List(ctx context.Context) ([]domain.Category, error)
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	if h.categories == nil {
		writeInternalError(w, "category service is not configured")
		return
	}
	categories, err := h.categories.List(r.Context())
	if err != nil {
		writeInternalError(w, "failed to list categories")
		return
	}
	data := make([]gen.Category, len(categories))
	for i, c := range categories {
		data[i] = gen.Category{
			Id:          c.ID,
			Slug:        c.Slug,
			DisplayName: c.DisplayName,
			Kind:        c.Kind,
			IsSystem:    c.IsSystem,
		}
	}
	writeJSON(w, http.StatusOK, gen.CategoryListResponse{Data: data})
}

func (h *Handler) GetClassificationQueue(w http.ResponseWriter, r *http.Request) {
	if h.classify == nil {
		writeInternalError(w, "classify service is not configured")
		return
	}
	items, err := h.classify.ListQueue(r.Context())
	if err != nil {
		writeInternalError(w, "failed to list classification queue")
		return
	}
	data := make([]gen.ClassificationQueueItem, len(items))
	for i, item := range items {
		data[i] = toAPIClassificationQueueItem(item)
	}
	writeJSON(w, http.StatusOK, gen.ClassificationQueueListResponse{Data: data})
}

func (h *Handler) PostClassificationApplySuggestions(w http.ResponseWriter, r *http.Request) {
	if h.classify == nil {
		writeInternalError(w, "classify service is not configured")
		return
	}
	result, err := h.classify.ApplySuggestions(r.Context())
	if err != nil {
		writeInternalError(w, "failed to apply classification suggestions")
		return
	}
	samples := make([]gen.ClassificationApplySuggestionSample, len(result.Samples))
	for i, s := range result.Samples {
		samples[i] = gen.ClassificationApplySuggestionSample{
			TransactionId: s.TransactionID,
			CategorySlug:  s.CategorySlug,
			Pattern:       s.Pattern,
			Confidence:    s.Confidence,
		}
	}
	writeJSON(w, http.StatusOK, gen.ClassificationApplySuggestionsResponse{
		Applied: result.Applied,
		Skipped: result.Skipped,
		Samples: samples,
	})
}

func (h *Handler) PostClassificationCorrect(
	w http.ResponseWriter,
	r *http.Request,
	transactionId openapi_types.UUID,
) {
	if h.classify == nil {
		writeInternalError(w, "classify service is not configured")
		return
	}

	var body gen.ClassificationCorrectRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	slug := strings.TrimSpace(body.CategorySlug)
	if slug == "" {
		writeValidationError(w, "category_slug is required")
		return
	}
	apply := false
	if body.ApplyToMerchant != nil {
		apply = *body.ApplyToMerchant
	}

	result, err := h.classify.Correct(r.Context(), transactionId, domain.ClassifyCorrectOptions{
		CategorySlug:    slug,
		ApplyToMerchant: apply,
	})
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not found"):
			writeNotFoundError(w, msg)
		case strings.Contains(msg, "unknown category"):
			writeValidationError(w, msg)
		default:
			writeInternalError(w, "failed to correct classification")
		}
		return
	}

	resp := gen.ClassificationCorrectResponse{
		TransactionId: result.TransactionID,
		CategorySlug:  result.CategorySlug,
		RuleCreated:   result.RuleCreated,
	}
	if result.MerchantID != nil {
		id := *result.MerchantID
		resp.MerchantId = &id
	}
	writeJSON(w, http.StatusOK, resp)
}

func toAPIClassificationQueueItem(item domain.ClassificationQueueItem) gen.ClassificationQueueItem {
	out := gen.ClassificationQueueItem{
		TransactionId: item.TransactionID,
		BookingDate:   openapi_types.Date{Time: dateOnly(item.BookingDate)},
		Amount:        item.Amount.StringFixed(2),
		Counterparty:  item.Counterparty,
		Purpose:       item.Purpose,
		BookingText:   item.BookingText,
		CategorySlug:  item.CategorySlug,
		CategoryName:  item.CategoryName,
		Source:        item.Source,
		Confidence:    item.Confidence,
		MerchantName:  item.MerchantName,
	}
	if item.MerchantID != nil {
		id := *item.MerchantID
		out.MerchantId = &id
	}
	return out
}
