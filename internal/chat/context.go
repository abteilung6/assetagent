package chat

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPageContext = errors.New("invalid chat page context")
	yyyyMMPattern         = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}$`)
)

type pageContextKey struct{}

// PageContext is a console viewing hint. Numbers must still come from tools.
type PageContext struct {
	Route      string
	BaselineID string
	ReviewID   string
	ForecastID string
	YYYYMM     string
	From       string
	To         string
	Tab        string
	Q          string
}

func (c PageContext) Empty() bool {
	return c.Route == "" &&
		c.BaselineID == "" &&
		c.ReviewID == "" &&
		c.ForecastID == "" &&
		c.YYYYMM == "" &&
		c.From == "" &&
		c.To == "" &&
		c.Tab == "" &&
		c.Q == ""
}

func WithPageContext(ctx context.Context, pageCtx PageContext) context.Context {
	if pageCtx.Empty() {
		return ctx
	}
	return context.WithValue(ctx, pageContextKey{}, pageCtx)
}

func PageContextFrom(ctx context.Context) PageContext {
	pageCtx, _ := ctx.Value(pageContextKey{}).(PageContext)
	return pageCtx
}

func ValidatePageContext(pageCtx PageContext) error {
	if pageCtx.Empty() {
		return nil
	}
	if pageCtx.Route != "" && len(pageCtx.Route) > 200 {
		return fmt.Errorf("%w: route too long", ErrInvalidPageContext)
	}
	if err := validateOptionalUUID("baseline_id", pageCtx.BaselineID); err != nil {
		return err
	}
	if err := validateOptionalUUID("review_id", pageCtx.ReviewID); err != nil {
		return err
	}
	if err := validateOptionalUUID("forecast_id", pageCtx.ForecastID); err != nil {
		return err
	}
	if pageCtx.YYYYMM != "" && !yyyyMMPattern.MatchString(pageCtx.YYYYMM) {
		return fmt.Errorf("%w: yyyy_mm must be YYYY-MM", ErrInvalidPageContext)
	}
	if pageCtx.From != "" {
		if _, err := time.Parse("2006-01-02", pageCtx.From); err != nil {
			return fmt.Errorf("%w: invalid from date", ErrInvalidPageContext)
		}
	}
	if pageCtx.To != "" {
		if _, err := time.Parse("2006-01-02", pageCtx.To); err != nil {
			return fmt.Errorf("%w: invalid to date", ErrInvalidPageContext)
		}
	}
	if pageCtx.From != "" && pageCtx.To != "" {
		from, _ := time.Parse("2006-01-02", pageCtx.From)
		to, _ := time.Parse("2006-01-02", pageCtx.To)
		if to.Before(from) {
			return fmt.Errorf("%w: to must be on or after from", ErrInvalidPageContext)
		}
	}
	if len(pageCtx.Tab) > 64 {
		return fmt.Errorf("%w: tab too long", ErrInvalidPageContext)
	}
	if len(pageCtx.Q) > 200 {
		return fmt.Errorf("%w: q too long", ErrInvalidPageContext)
	}
	return nil
}

func FormatContextAppendix(pageCtx PageContext) string {
	if pageCtx.Empty() {
		return ""
	}

	var b strings.Builder
	b.WriteString("User viewing (hint only — use tools for numbers; context is not a source of facts):\n")
	writeField(&b, "route", pageCtx.Route)
	writeField(&b, "baseline_id", pageCtx.BaselineID)
	writeField(&b, "review_id", pageCtx.ReviewID)
	writeField(&b, "forecast_id", pageCtx.ForecastID)
	writeField(&b, "yyyy_mm", pageCtx.YYYYMM)
	writeField(&b, "from", pageCtx.From)
	writeField(&b, "to", pageCtx.To)
	writeField(&b, "tab", pageCtx.Tab)
	writeField(&b, "q", pageCtx.Q)
	return strings.TrimSpace(b.String())
}

func writeField(b *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "- %s: %s\n", key, value)
}

func validateOptionalUUID(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("%w: invalid %s", ErrInvalidPageContext, field)
	}
	return nil
}
