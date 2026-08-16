package service_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSession_IssueResolveLogout(t *testing.T) {
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

	user, err := auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: "Ada"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	household, err := auth.CreateHousehold(ctx, "Ada household")
	if err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	if _, err := auth.CreateMembership(ctx, household.ID, user.ID, domain.MembershipRoleOwner); err != nil {
		t.Fatalf("CreateMembership: %v", err)
	}
	if _, err := auth.UpsertAuthIdentity(ctx, user.ID, domain.AuthProviderGoogle, "sub-1", "ada@example.com", true); err != nil {
		t.Fatalf("UpsertAuthIdentity: %v", err)
	}

	ip := netip.MustParseAddr("127.0.0.1")
	raw, session, err := sessions.IssueSession(ctx, user.ID, "test-agent", &ip)
	if err != nil {
		t.Fatalf("IssueSession: %v", err)
	}
	if raw == "" || session.ID == uuid.Nil {
		t.Fatalf("IssueSession returned empty token/session: raw=%q session=%+v", raw, session)
	}
	if len(session.TokenHash) != 32 {
		t.Fatalf("token hash len = %d, want 32", len(session.TokenHash))
	}

	gotUser, gotHousehold, gotMembership, gotSession, err := sessions.ResolveSession(ctx, raw)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if gotUser.ID != user.ID || gotHousehold.ID != household.ID {
		t.Fatalf("resolve user/household = %v/%v, want %v/%v", gotUser.ID, gotHousehold.ID, user.ID, household.ID)
	}
	if gotMembership.Role != domain.MembershipRoleOwner {
		t.Fatalf("membership role = %q", gotMembership.Role)
	}
	if gotSession.ID != session.ID {
		t.Fatalf("session id = %v, want %v", gotSession.ID, session.ID)
	}

	meUser, meHousehold, meMembership, email, err := sessions.LoadMe(ctx, user.ID)
	if err != nil {
		t.Fatalf("LoadMe: %v", err)
	}
	if meUser.ID != user.ID || meHousehold.ID != household.ID || meMembership.Role != domain.MembershipRoleOwner {
		t.Fatalf("LoadMe = %+v %+v %+v", meUser, meHousehold, meMembership)
	}
	if email != "ada@example.com" {
		t.Fatalf("email = %q, want ada@example.com", email)
	}

	updated, _, _, _, err := sessions.UpdatePreferredLocale(ctx, user.ID, domain.LocaleEN)
	if err != nil {
		t.Fatalf("UpdatePreferredLocale: %v", err)
	}
	if updated.PreferredLocale != domain.LocaleEN {
		t.Fatalf("preferred_locale = %q, want en", updated.PreferredLocale)
	}
	_, _, _, _, err = sessions.UpdatePreferredLocale(ctx, user.ID, "fr")
	if !errors.Is(err, service.ErrInvalidLocale) {
		t.Fatalf("UpdatePreferredLocale fr = %v, want ErrInvalidLocale", err)
	}

	if err := sessions.Logout(ctx, session.ID); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	_, _, _, _, err = sessions.ResolveSession(ctx, raw)
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("ResolveSession after logout = %v, want ErrUnauthorized", err)
	}
}

func TestSession_RejectsInvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	sessions := service.NewSession(repository.NewAuth(pool), service.SessionConfig{})

	_, _, _, _, err := sessions.ResolveSession(ctx, "not-a-real-token")
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("ResolveSession = %v, want ErrUnauthorized", err)
	}
}

func startMigratedPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
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
