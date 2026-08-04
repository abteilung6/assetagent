package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/abteilung6/assetagent/internal/classify"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Classify struct {
	pool *pgxpool.Pool
}

func NewClassify(pool *pgxpool.Pool) *Classify {
	return &Classify{pool: pool}
}

type PatternRuleImportResult struct {
	Upserted int
	Skipped  int
}

type ApplySuggestionsResult struct {
	Applied int
	Skipped int
	Samples []ApplySuggestionSample
}

type ApplySuggestionSample struct {
	TransactionID uuid.UUID
	CategorySlug  string
	Pattern       string
	Confidence    string
}

type CategorySuggestion struct {
	TransactionID   uuid.UUID
	BookingDate     string
	Amount          string
	Counterparty    string
	Purpose         string
	CurrentSlug     string
	SuggestedSlug   string
	MatchedPattern  string
	Confidence      string
	AutoApplicable  bool
}

func (s *Classify) Run(ctx context.Context) (domain.ClassifyRunResult, error) {
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return domain.ClassifyRunResult{}, err
	}
	q := sqldb.New(s.pool)

	if _, err := NewMerchants(s.pool).Rebuild(ctx); err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("merchant rebuild: %w", err)
	}

	categories, err := q.ListCategories(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list categories: %w", err)
	}
	bySlug := make(map[string]uuid.UUID, len(categories))
	slugByID := make(map[uuid.UUID]string, len(categories))
	for _, c := range categories {
		bySlug[c.Slug] = c.ID
		slugByID[c.ID] = c.Slug
	}

	rules, err := q.ListClassificationRules(ctx, householdID)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list rules: %w", err)
	}
	ruleByMerchant := make(map[uuid.UUID]sqldb.ClassificationRule, len(rules))
	patternRules := make([]classify.PatternRule, 0)
	for _, rule := range rules {
		if rule.MerchantID.Valid {
			ruleByMerchant[uuid.UUID(rule.MerchantID.Bytes)] = rule
			continue
		}
		if !rule.Pattern.Valid || rule.Pattern.String == "" {
			continue
		}
		slug := slugByID[rule.CategoryID]
		if slug == "" {
			continue
		}
		patternRules = append(patternRules, classify.PatternRule{
			Pattern:    rule.Pattern.String,
			Slug:       slug,
			Priority:   int(rule.Priority),
			Confidence: rule.Confidence,
		})
	}

	transferRows, err := q.ListConfirmedTransferTransactionIDs(ctx, householdID)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list transfers: %w", err)
	}
	transfers := make(map[uuid.UUID]struct{}, len(transferRows))
	for _, id := range transferRows {
		transfers[id] = struct{}{}
	}

	txs, err := q.ListTransactionsForClassify(ctx, householdID)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list txs: %w", err)
	}

	result := domain.ClassifyRunResult{
		Transactions: len(txs),
		BySource:     map[string]int64{},
		ByCategory:   map[string]int64{},
	}

	for _, tx := range txs {
		_, isTransfer := transfers[tx.ID]
		merchantPattern := ""
		var merchantID pgtype.UUID

		if label, ok := classify.NormalizeMerchantLabel(tx.Counterparty, tx.Purpose); ok {
			merchantPattern = label.Pattern
			alias, err := q.GetMerchantAlias(ctx, sqldb.GetMerchantAliasParams{
				MatchType:   domain.MerchantMatchNormalized,
				Pattern:     label.Pattern,
				HouseholdID: householdID,
			})
			if err == nil {
				merchantID = pgtype.UUID{Bytes: alias.MerchantID, Valid: true}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return domain.ClassifyRunResult{}, fmt.Errorf("get alias: %w", err)
			}
		}

		var (
			categoryID uuid.UUID
			source     string
			confidence string
			slug       string
		)

		if merchantID.Valid {
			if rule, ok := ruleByMerchant[uuid.UUID(merchantID.Bytes)]; ok {
				categoryID = rule.CategoryID
				source = domain.ClassificationSourceUserRule
				confidence = domain.ClassificationConfidenceHigh
				slug = slugByID[rule.CategoryID]
			}
		}

		if source == "" {
			if match := classify.MatchPattern(tx.Counterparty, tx.Purpose, patternRules); match != nil {
				var ok bool
				categoryID, ok = bySlug[match.Slug]
				if !ok {
					return domain.ClassifyRunResult{}, fmt.Errorf("missing category slug %q", match.Slug)
				}
				slug = match.Slug
				source = domain.ClassificationSourceExactRule
				confidence = match.Confidence
			}
		}

		if source == "" {
			slug, source, confidence = classify.SuggestCategory(tx.Amount, merchantPattern, isTransfer)
			var ok bool
			categoryID, ok = bySlug[slug]
			if !ok {
				return domain.ClassifyRunResult{}, fmt.Errorf("missing category slug %q", slug)
			}
		}

		row, err := q.UpsertTransactionClassification(ctx, sqldb.UpsertTransactionClassificationParams{
			TransactionID:    tx.ID,
			CategoryID:       categoryID,
			MerchantID:       merchantID,
			Source:           source,
			Confidence:       confidence,
			AlgorithmVersion: domain.ClassifyAlgorithmVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				result.SkippedUser++
				continue
			}
			return domain.ClassifyRunResult{}, fmt.Errorf("upsert classification: %w", err)
		}
		result.Upserted++
		result.BySource[row.Source]++
		if slug == "" {
			slug = slugByID[row.CategoryID]
		}
		result.ByCategory[slug]++
	}

	return result, nil
}

func (s *Classify) Correct(
	ctx context.Context,
	txID uuid.UUID,
	opts domain.ClassifyCorrectOptions,
) (domain.ClassifyCorrectResult, error) {
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return domain.ClassifyCorrectResult{}, err
	}
	q := sqldb.New(s.pool)

	slug := opts.CategorySlug
	category, err := q.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, fmt.Errorf("unknown category %q", slug)
		}
		return domain.ClassifyCorrectResult{}, err
	}

	tx, err := q.GetTransactionForClassify(ctx, sqldb.GetTransactionForClassifyParams{
		ID:          txID,
		HouseholdID: householdID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, fmt.Errorf("transaction not found")
		}
		return domain.ClassifyCorrectResult{}, err
	}

	var merchantID pgtype.UUID
	if label, ok := classify.NormalizeMerchantLabel(tx.Counterparty, tx.Purpose); ok {
		if _, err := NewMerchants(s.pool).Rebuild(ctx); err != nil {
			return domain.ClassifyCorrectResult{}, err
		}
		alias, err := q.GetMerchantAlias(ctx, sqldb.GetMerchantAliasParams{
			MatchType:   domain.MerchantMatchNormalized,
			Pattern:     label.Pattern,
			HouseholdID: householdID,
		})
		if err == nil {
			merchantID = pgtype.UUID{Bytes: alias.MerchantID, Valid: true}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, err
		}
	}

	if _, err := q.ForceUpsertTransactionClassification(ctx, sqldb.ForceUpsertTransactionClassificationParams{
		TransactionID:    txID,
		CategoryID:       category.ID,
		MerchantID:       merchantID,
		Source:           domain.ClassificationSourceUserRule,
		Confidence:       domain.ClassificationConfidenceHigh,
		AlgorithmVersion: domain.ClassifyAlgorithmVersion,
	}); err != nil {
		return domain.ClassifyCorrectResult{}, err
	}

	result := domain.ClassifyCorrectResult{
		TransactionID: txID,
		CategorySlug:  slug,
	}

	if opts.ApplyToMerchant && merchantID.Valid {
		mid := uuid.UUID(merchantID.Bytes)
		result.MerchantID = &mid
		existing, err := q.GetClassificationRuleByMerchant(ctx, sqldb.GetClassificationRuleByMerchantParams{
			MerchantID:  merchantID,
			HouseholdID: householdID,
		})
		switch {
		case err == nil:
			if _, err := q.UpdateClassificationRuleCategory(ctx, sqldb.UpdateClassificationRuleCategoryParams{
				ID:                       existing.ID,
				HouseholdID:              householdID,
				CategoryID:               category.ID,
				CreatedFromTransactionID: pgtype.UUID{Bytes: txID, Valid: true},
				Priority:                 10,
			}); err != nil {
				return domain.ClassifyCorrectResult{}, err
			}
			result.RuleCreated = true
		case errors.Is(err, pgx.ErrNoRows):
			if _, err := q.CreateClassificationRule(ctx, sqldb.CreateClassificationRuleParams{
				HouseholdID:              householdID,
				Priority:                 10,
				MerchantID:               merchantID,
				Pattern:                  pgtype.Text{},
				CategoryID:               category.ID,
				CreatedFromTransactionID: pgtype.UUID{Bytes: txID, Valid: true},
				Confidence:               domain.ClassificationConfidenceHigh,
				IsSystem:                 false,
			}); err != nil {
				return domain.ClassifyCorrectResult{}, err
			}
			result.RuleCreated = true
		default:
			return domain.ClassifyCorrectResult{}, err
		}

		if _, err := q.UpdateMerchantDefaultCategory(ctx, sqldb.UpdateMerchantDefaultCategoryParams{
			ID:                mid,
			HouseholdID:       householdID,
			DefaultCategoryID: pgtype.UUID{Bytes: category.ID, Valid: true},
		}); err != nil {
			return domain.ClassifyCorrectResult{}, err
		}
	}

	return result, nil
}

func (s *Classify) ListQueue(ctx context.Context) ([]domain.ClassificationQueueItem, error) {
	// Keep the inbox populated after imports without a separate classify step.
	if _, err := s.Run(ctx); err != nil {
		return nil, fmt.Errorf("classify before queue: %w", err)
	}

	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := sqldb.New(s.pool).ListClassificationQueue(ctx, householdID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.ClassificationQueueItem, len(rows))
	for i, row := range rows {
		item := domain.ClassificationQueueItem{
			TransactionID: row.TransactionID,
			BookingDate:   row.BookingDate.Time,
			Amount:        row.Amount,
			Counterparty:  row.Counterparty,
			Purpose:       row.Purpose,
			BookingText:   row.BookingText,
			CategorySlug:  row.CategorySlug,
			CategoryName:  row.CategoryName,
			Source:        row.Source,
			Confidence:    row.Confidence,
			MerchantName:  row.MerchantName,
		}
		if row.MerchantID.Valid {
			id := uuid.UUID(row.MerchantID.Bytes)
			item.MerchantID = &id
		}
		out[i] = item
	}
	return out, nil
}

func (s *Classify) loadPatternRules(ctx context.Context) ([]classify.PatternRule, error) {
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	q := sqldb.New(s.pool)
	categories, err := q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	slugByID := make(map[uuid.UUID]string, len(categories))
	for _, c := range categories {
		slugByID[c.ID] = c.Slug
	}
	rules, err := q.ListClassificationRules(ctx, householdID)
	if err != nil {
		return nil, err
	}
	out := make([]classify.PatternRule, 0)
	for _, rule := range rules {
		if rule.MerchantID.Valid || !rule.Pattern.Valid || rule.Pattern.String == "" {
			continue
		}
		slug := slugByID[rule.CategoryID]
		if slug == "" {
			continue
		}
		out = append(out, classify.PatternRule{
			Pattern:    rule.Pattern.String,
			Slug:       slug,
			Priority:   int(rule.Priority),
			Confidence: rule.Confidence,
		})
	}
	return out, nil
}

// SuggestCategories returns queue items with pattern-based suggestions.
func (s *Classify) SuggestCategories(ctx context.Context) ([]CategorySuggestion, error) {
	queue, err := s.ListQueue(ctx)
	if err != nil {
		return nil, err
	}
	patterns, err := s.loadPatternRules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CategorySuggestion, 0, len(queue))
	for _, item := range queue {
		match := classify.MatchPattern(item.Counterparty, item.Purpose, patterns)
		sug := CategorySuggestion{
			TransactionID: item.TransactionID,
			BookingDate:   item.BookingDate.Format("2006-01-02"),
			Amount:        item.Amount.StringFixed(2),
			Counterparty:  item.Counterparty,
			Purpose:       item.Purpose,
			CurrentSlug:   item.CategorySlug,
		}
		if match != nil {
			sug.SuggestedSlug = match.Slug
			sug.MatchedPattern = match.Pattern
			sug.Confidence = match.Confidence
			sug.AutoApplicable = classify.ShouldAutoApply(match)
		}
		out = append(out, sug)
	}
	return out, nil
}

// ApplySuggestions applies high/medium pattern matches on the current queue.
func (s *Classify) ApplySuggestions(ctx context.Context) (ApplySuggestionsResult, error) {
	suggestions, err := s.SuggestCategories(ctx)
	if err != nil {
		return ApplySuggestionsResult{}, err
	}

	result := ApplySuggestionsResult{
		Samples: make([]ApplySuggestionSample, 0),
	}
	for _, sug := range suggestions {
		if !sug.AutoApplicable || sug.SuggestedSlug == "" {
			result.Skipped++
			continue
		}
		if sug.SuggestedSlug == sug.CurrentSlug && sug.Confidence == domain.ClassificationConfidenceHigh {
			// Already classified by pattern; still promote to user_rule + merchant memory.
		}
		_, err := s.Correct(ctx, sug.TransactionID, domain.ClassifyCorrectOptions{
			CategorySlug:    sug.SuggestedSlug,
			ApplyToMerchant: true,
		})
		if err != nil {
			return ApplySuggestionsResult{}, fmt.Errorf("apply %s: %w", sug.TransactionID, err)
		}
		result.Applied++
		if len(result.Samples) < 10 {
			result.Samples = append(result.Samples, ApplySuggestionSample{
				TransactionID: sug.TransactionID,
				CategorySlug:  sug.SuggestedSlug,
				Pattern:       sug.MatchedPattern,
				Confidence:    sug.Confidence,
			})
		}
	}
	return result, nil
}

// ImportPatternRulesCSV upserts system pattern rules from a CSV file.
func (s *Classify) ImportPatternRulesCSV(ctx context.Context, path string) (PatternRuleImportResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return PatternRuleImportResult{}, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return PatternRuleImportResult{}, fmt.Errorf("read csv header: %w", err)
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, want := range []string{"pattern", "category_slug", "priority", "confidence"} {
		if _, ok := col[want]; !ok {
			return PatternRuleImportResult{}, fmt.Errorf("csv missing column %q", want)
		}
	}

	q := sqldb.New(s.pool)
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return PatternRuleImportResult{}, err
	}
	categories, err := q.ListCategories(ctx)
	if err != nil {
		return PatternRuleImportResult{}, err
	}
	bySlug := make(map[string]uuid.UUID, len(categories))
	for _, c := range categories {
		bySlug[c.Slug] = c.ID
	}

	var result PatternRuleImportResult
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return PatternRuleImportResult{}, fmt.Errorf("read csv row: %w", err)
		}
		pattern := strings.TrimSpace(row[col["pattern"]])
		slug := strings.TrimSpace(row[col["category_slug"]])
		priorityRaw := strings.TrimSpace(row[col["priority"]])
		confidence := strings.TrimSpace(row[col["confidence"]])
		if pattern == "" || slug == "" {
			result.Skipped++
			continue
		}
		categoryID, ok := bySlug[slug]
		if !ok {
			return PatternRuleImportResult{}, fmt.Errorf("unknown category_slug %q", slug)
		}
		priority, err := strconv.Atoi(priorityRaw)
		if err != nil {
			return PatternRuleImportResult{}, fmt.Errorf("invalid priority %q: %w", priorityRaw, err)
		}
		switch confidence {
		case domain.ClassificationConfidenceHigh,
			domain.ClassificationConfidenceMedium,
			domain.ClassificationConfidenceLow:
		default:
			return PatternRuleImportResult{}, fmt.Errorf("invalid confidence %q", confidence)
		}

		if _, err := q.UpsertSystemPatternRule(ctx, sqldb.UpsertSystemPatternRuleParams{
			HouseholdID: householdID,
			Priority:    int32(priority),
			Pattern:     pgtype.Text{String: pattern, Valid: true},
			CategoryID:  categoryID,
			Confidence:  confidence,
		}); err != nil {
			return PatternRuleImportResult{}, fmt.Errorf("upsert pattern %q: %w", pattern, err)
		}
		result.Upserted++
	}
	return result, nil
}
