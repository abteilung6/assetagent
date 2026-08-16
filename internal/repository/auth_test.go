package repository_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestAuth_userHouseholdSessionRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := startMigratedPool(ctx, t)
	auth := repository.NewAuth(pool)

	user, err := auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: "Ada"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == uuid.Nil || user.DisplayName != "Ada" {
		t.Fatalf("CreateUser result = %+v", user)
	}
	if user.PreferredLocale != domain.LocaleDE {
		t.Fatalf("preferred_locale = %q, want %q", user.PreferredLocale, domain.LocaleDE)
	}

	gotUser, err := auth.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if gotUser.ID != user.ID {
		t.Fatalf("GetUser ID = %v, want %v", gotUser.ID, user.ID)
	}

	profiled, err := auth.CreateUser(ctx, repository.CreateUserInput{
		DisplayName:     "Grace Hopper",
		GivenName:       "Grace",
		PictureURL:      "https://example.com/grace.png",
		PreferredLocale: domain.LocaleEN,
	})
	if err != nil {
		t.Fatalf("CreateUser profiled: %v", err)
	}
	if profiled.GivenName != "Grace" || profiled.PictureURL != "https://example.com/grace.png" || profiled.PreferredLocale != domain.LocaleEN {
		t.Fatalf("profiled user = %+v", profiled)
	}
	refreshed, err := auth.UpdateUserGoogleProfile(ctx, profiled.ID, "Gracie", "https://example.com/grace-new.png")
	if err != nil {
		t.Fatalf("UpdateUserGoogleProfile: %v", err)
	}
	if refreshed.GivenName != "Grace" {
		t.Fatalf("given_name should stay Grace, got %q", refreshed.GivenName)
	}
	if refreshed.PictureURL != "https://example.com/grace-new.png" {
		t.Fatalf("picture_url = %q", refreshed.PictureURL)
	}
	filled, err := auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: "No Given"})
	if err != nil {
		t.Fatalf("CreateUser bare: %v", err)
	}
	filled, err = auth.UpdateUserGoogleProfile(ctx, filled.ID, "Nora", "https://example.com/nora.png")
	if err != nil {
		t.Fatalf("UpdateUserGoogleProfile fill: %v", err)
	}
	if filled.GivenName != "Nora" || filled.PictureURL != "https://example.com/nora.png" {
		t.Fatalf("filled profile = %+v", filled)
	}

	identity, err := auth.UpsertAuthIdentity(ctx, user.ID, domain.AuthProviderGoogle, "google-sub-1", "ada@example.com", true)
	if err != nil {
		t.Fatalf("UpsertAuthIdentity: %v", err)
	}
	if identity.ProviderSubject != "google-sub-1" || !identity.EmailVerified {
		t.Fatalf("identity = %+v", identity)
	}

	gotIdentity, err := auth.GetAuthIdentityByProviderSubject(ctx, domain.AuthProviderGoogle, "google-sub-1")
	if err != nil {
		t.Fatalf("GetAuthIdentityByProviderSubject: %v", err)
	}
	if gotIdentity.UserID != user.ID {
		t.Fatalf("identity user = %v, want %v", gotIdentity.UserID, user.ID)
	}

	updated, err := auth.UpsertAuthIdentity(ctx, user.ID, domain.AuthProviderGoogle, "google-sub-1", "ada2@example.com", true)
	if err != nil {
		t.Fatalf("UpsertAuthIdentity update: %v", err)
	}
	if updated.ID != identity.ID || updated.Email != "ada2@example.com" {
		t.Fatalf("upsert update = %+v", updated)
	}

	// Migration 00019 inserts an unclaimed "Local seed" household.
	seed, err := auth.GetUnclaimedSeedHousehold(ctx)
	if err != nil {
		t.Fatalf("GetUnclaimedSeedHousehold: %v", err)
	}
	if seed.Name != domain.SeedHouseholdName || seed.ClaimedAt != nil {
		t.Fatalf("seed household = %+v", seed)
	}

	byName, err := auth.GetHouseholdByName(ctx, domain.SeedHouseholdName)
	if err != nil {
		t.Fatalf("GetHouseholdByName: %v", err)
	}
	if byName.ID != seed.ID {
		t.Fatalf("GetHouseholdByName = %v, want %v", byName.ID, seed.ID)
	}

	extra, err := auth.CreateHousehold(ctx, "Extra household")
	if err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	if extra.ClaimedAt != nil {
		t.Fatalf("new household should be unclaimed, got claimed_at=%v", extra.ClaimedAt)
	}

	membership, err := auth.CreateMembership(ctx, seed.ID, user.ID, domain.MembershipRoleOwner)
	if err != nil {
		t.Fatalf("CreateMembership: %v", err)
	}
	if membership.Role != domain.MembershipRoleOwner {
		t.Fatalf("membership role = %q", membership.Role)
	}

	memberships, err := auth.GetMembershipsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetMembershipsByUser: %v", err)
	}
	if len(memberships) != 1 || memberships[0].HouseholdID != seed.ID {
		t.Fatalf("memberships = %+v", memberships)
	}

	claimed, err := auth.ClaimHousehold(ctx, seed.ID)
	if err != nil {
		t.Fatalf("ClaimHousehold: %v", err)
	}
	if claimed.ClaimedAt == nil {
		t.Fatal("ClaimHousehold left claimed_at nil")
	}
	_, err = auth.ClaimHousehold(ctx, seed.ID)
	if err == nil {
		t.Fatal("second ClaimHousehold should fail (already claimed)")
	}
	if err != pgx.ErrNoRows {
		t.Fatalf("second ClaimHousehold err = %v, want ErrNoRows", err)
	}

	// Extra household remains unclaimed, so seed lookup still finds a row.
	stillUnclaimed, err := auth.GetUnclaimedSeedHousehold(ctx)
	if err != nil {
		t.Fatalf("GetUnclaimedSeedHousehold after claim: %v", err)
	}
	if stillUnclaimed.ID != extra.ID {
		t.Fatalf("unclaimed after seed claim = %v, want extra %v", stillUnclaimed.ID, extra.ID)
	}
	if _, err := auth.ClaimHousehold(ctx, extra.ID); err != nil {
		t.Fatalf("ClaimHousehold extra: %v", err)
	}
	_, err = auth.GetUnclaimedSeedHousehold(ctx)
	if err != pgx.ErrNoRows {
		t.Fatalf("GetUnclaimedSeedHousehold after all claimed = %v, want ErrNoRows", err)
	}

	token := []byte("opaque-session-token")
	sum := sha256.Sum256(token)
	now := time.Now().UTC().Truncate(time.Millisecond)
	session, err := auth.CreateSession(ctx, repository.CreateSessionInput{
		UserID:            user.ID,
		TokenHash:         sum[:],
		ExpiresAt:         now.Add(24 * time.Hour),
		AbsoluteExpiresAt: now.Add(72 * time.Hour),
		UserAgent:         "test-agent",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	gotSession, err := auth.GetSessionByTokenHash(ctx, sum[:])
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if gotSession.ID != session.ID || gotSession.UserID != user.ID {
		t.Fatalf("session = %+v", gotSession)
	}

	touched, err := auth.TouchSession(ctx, session.ID, now.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("TouchSession: %v", err)
	}
	if !touched.ExpiresAt.After(session.ExpiresAt) {
		t.Fatalf("TouchSession expires_at = %v, want after %v", touched.ExpiresAt, session.ExpiresAt)
	}

	revoked, err := auth.RevokeSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeSession left revoked_at nil")
	}
	_, err = auth.RevokeSession(ctx, session.ID)
	if err != pgx.ErrNoRows {
		t.Fatalf("second RevokeSession = %v, want ErrNoRows", err)
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

	waitForDB(ctx, t, connStr)

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
