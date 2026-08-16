package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/api/middleware"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type testSessionFixture struct {
	pool     *pgxpool.Pool
	sessions *service.SessionService
	router   chi.Router
	auth     *repository.Auth
}

func setupAuthFixture(t *testing.T) *testSessionFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startAuthPool(ctx, t)
	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{
		CookieName: "session",
		Idle:       2 * time.Hour,
		Absolute:   24 * time.Hour,
	})

	router := chi.NewRouter()
	router.Use(middleware.ResolveSessionMiddleware(sessions))
	router.Use(middleware.RequireAuthMiddleware)
	gen.HandlerWithOptions(
		handler.New(noopList{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessions),
		gen.ChiServerOptions{
			BaseRouter:       router,
			ErrorHandlerFunc: handler.APIErrorHandler,
		},
	)

	return &testSessionFixture{
		pool:     pool,
		sessions: sessions,
		router:   router,
		auth:     auth,
	}
}

// CreateTestSession inserts a user+membership+session and returns the session cookie.
func CreateTestSession(t *testing.T, f *testSessionFixture, displayName, email string) *http.Cookie {
	t.Helper()
	ctx := context.Background()

	user, err := f.auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: displayName})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	household, err := f.auth.CreateHousehold(ctx, displayName+" household")
	if err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	if _, err := f.auth.CreateMembership(ctx, household.ID, user.ID, domain.MembershipRoleOwner); err != nil {
		t.Fatalf("CreateMembership: %v", err)
	}
	if email != "" {
		if _, err := f.auth.UpsertAuthIdentity(ctx, user.ID, domain.AuthProviderGoogle, "sub-"+user.ID.String(), email, true); err != nil {
			t.Fatalf("UpsertAuthIdentity: %v", err)
		}
	}

	raw, _, err := f.sessions.IssueSession(ctx, user.ID, "test-agent", nil)
	if err != nil {
		t.Fatalf("IssueSession: %v", err)
	}
	return &http.Cookie{
		Name:  f.sessions.CookieName(),
		Value: raw,
		Path:  "/",
	}
}

func TestGetMe_unauthorized(t *testing.T) {
	f := setupAuthFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetMe_withCookie(t *testing.T) {
	f := setupAuthFixture(t)
	cookie := CreateTestSession(t, f, "Ada", "ada@example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var resp gen.MeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.User.DisplayName != "Ada" {
		t.Fatalf("display_name = %q, want Ada", resp.User.DisplayName)
	}
	if resp.User.PreferredLocale != gen.De {
		t.Fatalf("preferred_locale = %q, want de", resp.User.PreferredLocale)
	}
	if resp.User.Email == nil || string(*resp.User.Email) != "ada@example.com" {
		t.Fatalf("email = %v, want ada@example.com", resp.User.Email)
	}
	if resp.Household.Name == "" || resp.Membership.Role != gen.Owner {
		t.Fatalf("household/membership = %+v / %+v", resp.Household, resp.Membership)
	}
}

func TestGetMe_includesGoogleProfileFields(t *testing.T) {
	f := setupAuthFixture(t)
	ctx := context.Background()

	user, err := f.auth.CreateUser(ctx, repository.CreateUserInput{
		DisplayName:     "Ada Lovelace",
		GivenName:       "Ada",
		PictureURL:      "https://example.com/ada.png",
		PreferredLocale: domain.LocaleEN,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	household, err := f.auth.CreateHousehold(ctx, "Ada household")
	if err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	if _, err := f.auth.CreateMembership(ctx, household.ID, user.ID, domain.MembershipRoleOwner); err != nil {
		t.Fatalf("CreateMembership: %v", err)
	}
	raw, _, err := f.sessions.IssueSession(ctx, user.ID, "test-agent", nil)
	if err != nil {
		t.Fatalf("IssueSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: f.sessions.CookieName(), Value: raw, Path: "/"})
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var resp gen.MeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.User.PreferredLocale != gen.En {
		t.Fatalf("preferred_locale = %q, want en", resp.User.PreferredLocale)
	}
	if resp.User.GivenName == nil || *resp.User.GivenName != "Ada" {
		t.Fatalf("given_name = %v", resp.User.GivenName)
	}
	if resp.User.PictureUrl == nil || *resp.User.PictureUrl != "https://example.com/ada.png" {
		t.Fatalf("picture_url = %v", resp.User.PictureUrl)
	}
}

func TestPostLogout_clearsSession(t *testing.T) {
	f := setupAuthFixture(t)
	cookie := CreateTestSession(t, f, "Ada", "ada@example.com")

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	f.router.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body = %s", logoutRec.Code, logoutRec.Body.String())
	}

	cleared := false
	for _, c := range logoutRec.Result().Cookies() {
		if c.Name == f.sessions.CookieName() && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected cleared session cookie on logout response")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	f.router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401; body = %s", meRec.Code, meRec.Body.String())
	}
}

func startAuthPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("assetagent"),
		postgres.WithUsername("assetagent"),
		postgres.WithPassword("assetagent"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		pool, err := db.NewPool(ctx, connStr)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				break
			}
			pool.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := db.RunMigrations(connStr, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := db.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
