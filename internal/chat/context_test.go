package chat_test

import (
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat"
)

func TestValidatePageContext_acceptsValid(t *testing.T) {
	err := chat.ValidatePageContext(chat.PageContext{
		Route:      "/insights/months/2026-03",
		BaselineID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		YYYYMM:     "2026-03",
		From:       "2026-03-01",
		To:         "2026-03-31",
		Tab:        "overview",
	})
	if err != nil {
		t.Fatalf("ValidatePageContext() = %v", err)
	}
}

func TestValidatePageContext_rejectsBadUUID(t *testing.T) {
	err := chat.ValidatePageContext(chat.PageContext{BaselineID: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidatePageContext_rejectsBadYYYYMM(t *testing.T) {
	err := chat.ValidatePageContext(chat.PageContext{YYYYMM: "2026-3"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatContextAppendix_includesRouteAndDates(t *testing.T) {
	appendix := chat.FormatContextAppendix(chat.PageContext{
		Route:  "/insights/months/2026-03",
		YYYYMM: "2026-03",
		From:   "2026-03-01",
		To:     "2026-03-31",
	})
	if !strings.Contains(appendix, "use tools for numbers") {
		t.Fatalf("missing tools disclaimer: %q", appendix)
	}
	if !strings.Contains(appendix, "route: /insights/months/2026-03") {
		t.Fatalf("missing route: %q", appendix)
	}
	if !strings.Contains(appendix, "from: 2026-03-01") {
		t.Fatalf("missing from: %q", appendix)
	}
}

func TestFormatContextAppendix_empty(t *testing.T) {
	if got := chat.FormatContextAppendix(chat.PageContext{}); got != "" {
		t.Fatalf("got %q", got)
	}
}
