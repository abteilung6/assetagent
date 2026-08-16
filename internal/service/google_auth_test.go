package service_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
)

type stubExchanger struct {
	authURL     string
	rawIDToken  string
	exchangeErr error
	lastState   string
	lastNonce   string
	lastChal    string
	lastCode    string
	lastVer     string
}

func (s *stubExchanger) AuthCodeURL(state, nonce, codeChallenge string) string {
	s.lastState = state
	s.lastNonce = nonce
	s.lastChal = codeChallenge
	if s.authURL != "" {
		return s.authURL + "?state=" + state
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
}

func (s *stubExchanger) Exchange(_ context.Context, code, codeVerifier string) (string, error) {
	s.lastCode = code
	s.lastVer = codeVerifier
	if s.exchangeErr != nil {
		return "", s.exchangeErr
	}
	return s.rawIDToken, nil
}

type stubVerifier struct {
	claims service.GoogleIDTokenClaims
	err    error
}

func (s *stubVerifier) Verify(_ context.Context, _, expectedNonce string) (service.GoogleIDTokenClaims, error) {
	if s.err != nil {
		return service.GoogleIDTokenClaims{}, s.err
	}
	out := s.claims
	out.Nonce = expectedNonce
	return out, nil
}

func TestGoogleAuth_StartAndComplete_newUserClaimsSeed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{
		CookieName: "session",
		Idle:       2 * time.Hour,
		Absolute:   24 * time.Hour,
	})

	exchanger := &stubExchanger{rawIDToken: "fake-id-token"}
	verifier := &stubVerifier{claims: service.GoogleIDTokenClaims{
		Subject:       "google-sub-1",
		Email:         "ada@example.com",
		EmailVerified: true,
		Name:          "Ada Lovelace",
		GivenName:     "Ada",
		PictureURL:    "https://lh3.googleusercontent.com/a/ada",
		Locale:        "en-GB",
	}}
	google := service.NewGoogleAuth(auth, sessions, exchanger, verifier, service.GoogleAuthConfig{
		FrontendURL:       "http://localhost:5173",
		ClaimExistingData: true,
		Configured:        true,
	})

	authURL, err := google.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.Contains(authURL, exchanger.lastState) || exchanger.lastNonce == "" || exchanger.lastChal == "" {
		t.Fatalf("AuthCodeURL missing state/nonce/challenge: url=%q state=%q", authURL, exchanger.lastState)
	}

	raw, err := google.Complete(ctx, exchanger.lastState, "auth-code", "test-agent", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if raw == "" {
		t.Fatal("expected session token")
	}

	user, household, _, _, err := sessions.ResolveSession(ctx, raw)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if user.DisplayName != "Ada Lovelace" {
		t.Fatalf("display name = %q", user.DisplayName)
	}
	if user.GivenName != "Ada" {
		t.Fatalf("given name = %q, want Ada", user.GivenName)
	}
	if user.PictureURL != "https://lh3.googleusercontent.com/a/ada" {
		t.Fatalf("picture = %q", user.PictureURL)
	}
	if user.PreferredLocale != domain.LocaleEN {
		t.Fatalf("preferred_locale = %q, want en", user.PreferredLocale)
	}
	if household.Name != domain.SeedHouseholdName {
		t.Fatalf("household = %q, want seed claimed", household.Name)
	}
	if household.ClaimedAt == nil {
		t.Fatal("expected seed household claimed_at set")
	}

	// Second login refreshes picture but does not overwrite preferred_locale or given_name.
	verifier.claims.GivenName = "Adelaide"
	verifier.claims.PictureURL = "https://lh3.googleusercontent.com/a/ada-new"
	verifier.claims.Locale = "de-DE"
	exchanger2 := &stubExchanger{rawIDToken: "fake-id-token-2"}
	google2 := service.NewGoogleAuth(auth, sessions, exchanger2, verifier, service.GoogleAuthConfig{
		FrontendURL:       "http://localhost:5173",
		ClaimExistingData: true,
		Configured:        true,
	})
	if _, err := google2.Start(ctx); err != nil {
		t.Fatalf("Start 2: %v", err)
	}
	raw2, err := google2.Complete(ctx, exchanger2.lastState, "auth-code-2", "test-agent", nil)
	if err != nil {
		t.Fatalf("Complete 2: %v", err)
	}
	user2, household2, _, _, err := sessions.ResolveSession(ctx, raw2)
	if err != nil {
		t.Fatalf("ResolveSession 2: %v", err)
	}
	if user2.ID != user.ID || household2.ID != household.ID {
		t.Fatalf("second login created new user/household")
	}
	if user2.GivenName != "Ada" {
		t.Fatalf("given_name overwritten on return login: %q", user2.GivenName)
	}
	if user2.PictureURL != "https://lh3.googleusercontent.com/a/ada-new" {
		t.Fatalf("picture not refreshed: %q", user2.PictureURL)
	}
	if user2.PreferredLocale != domain.LocaleEN {
		t.Fatalf("preferred_locale overwritten on return login: %q", user2.PreferredLocale)
	}
}

func TestGoogleAuth_Complete_rejectsUnverifiedEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{CookieName: "session"})
	exchanger := &stubExchanger{rawIDToken: "tok"}
	verifier := &stubVerifier{claims: service.GoogleIDTokenClaims{
		Subject:       "sub-unverified",
		Email:         "x@example.com",
		EmailVerified: false,
	}}
	google := service.NewGoogleAuth(auth, sessions, exchanger, verifier, service.GoogleAuthConfig{Configured: true})

	if _, err := google.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err := google.Complete(ctx, exchanger.lastState, "code", "", nil)
	if !errors.Is(err, service.ErrEmailNotVerified) {
		t.Fatalf("err = %v, want ErrEmailNotVerified", err)
	}
}

func TestGoogleAuth_NewUserWithoutClaimCreatesEmptyHousehold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{CookieName: "session"})
	exchanger := &stubExchanger{rawIDToken: "tok"}
	verifier := &stubVerifier{claims: service.GoogleIDTokenClaims{
		Subject:       "sub-empty",
		Email:         "bob@example.com",
		EmailVerified: true,
		Name:          "Bob",
	}}
	google := service.NewGoogleAuth(auth, sessions, exchanger, verifier, service.GoogleAuthConfig{
		ClaimExistingData: false,
		Configured:        true,
	})

	if _, err := google.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	raw, err := google.Complete(ctx, exchanger.lastState, "code", "", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	_, household, _, _, err := sessions.ResolveSession(ctx, raw)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if household.Name == domain.SeedHouseholdName {
		t.Fatal("should not claim seed when AUTH_CLAIM_EXISTING_DATA is false")
	}
}

func TestGoogleAuthHandler_StartUnavailable(t *testing.T) {
	router := chi.NewRouter()
	gh := handler.NewGoogleAuthHandler(nil, nil)
	router.Get("/auth/google/start", gh.Start)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/start", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestGoogleAuthHandler_CallbackSetsCookie(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{
		CookieName: "session",
		Idle:       time.Hour,
		Absolute:   24 * time.Hour,
	})
	exchanger := &stubExchanger{rawIDToken: "tok"}
	verifier := &stubVerifier{claims: service.GoogleIDTokenClaims{
		Subject:       "sub-http",
		Email:         "c@example.com",
		EmailVerified: true,
		Name:          "C",
	}}
	google := service.NewGoogleAuth(auth, sessions, exchanger, verifier, service.GoogleAuthConfig{
		FrontendURL: "http://localhost:5173",
		Configured:  true,
	})
	if _, err := google.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	router := chi.NewRouter()
	gh := handler.NewGoogleAuthHandler(google, sessions)
	router.Get("/auth/google/callback", gh.Callback)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+exchanger.lastState+"&code=abc", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "http://localhost:5173/" {
		t.Fatalf("Location = %q", loc)
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session cookie")
	}
}

func TestResolveSessionCookieOptions_production(t *testing.T) {
	name, secure := service.ResolveSessionCookieOptions("production", "session", false)
	if name != "__Host-session" || !secure {
		t.Fatalf("got %q secure=%v", name, secure)
	}
	name, secure = service.ResolveSessionCookieOptions("development", "session", false)
	if name != "session" || secure {
		t.Fatalf("dev got %q secure=%v", name, secure)
	}
}

func TestSessionConfigFromEnv_production(t *testing.T) {
	cfg := service.SessionConfigFromEnv("production", "session", false, 1, 2)
	if cfg.CookieName != "__Host-session" || !cfg.Secure {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Idle != time.Hour || cfg.Absolute != 2*time.Hour {
		t.Fatalf("durations = %v / %v", cfg.Idle, cfg.Absolute)
	}
}
