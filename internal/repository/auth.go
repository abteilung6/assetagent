package repository

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Auth struct {
	queries sqldb.Querier
}

func NewAuth(pool *pgxpool.Pool) *Auth {
	return &Auth{queries: sqldb.New(pool)}
}

type CreateUserInput struct {
	DisplayName     string
	GivenName       string
	PictureURL      string
	PreferredLocale string
}

func (a *Auth) CreateUser(ctx context.Context, in CreateUserInput) (domain.User, error) {
	locale := in.PreferredLocale
	if !domain.IsSupportedLocale(locale) {
		locale = domain.LocaleDE
	}
	row, err := a.queries.CreateUser(ctx, sqldb.CreateUserParams{
		DisplayName:     in.DisplayName,
		GivenName:       textOrNull(in.GivenName),
		PictureUrl:      textOrNull(in.PictureURL),
		PreferredLocale: locale,
	})
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (a *Auth) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := a.queries.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

// UpdateUserGoogleProfile refreshes picture_url and fills given_name only when unset.
func (a *Auth) UpdateUserGoogleProfile(ctx context.Context, userID uuid.UUID, givenName, pictureURL string) (domain.User, error) {
	row, err := a.queries.UpdateUserGoogleProfile(ctx, sqldb.UpdateUserGoogleProfileParams{
		ID:         userID,
		GivenName:  strings.TrimSpace(givenName),
		PictureUrl: strings.TrimSpace(pictureURL),
	})
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (a *Auth) UpdateUserPreferredLocale(ctx context.Context, userID uuid.UUID, locale string) (domain.User, error) {
	if !domain.IsSupportedLocale(locale) {
		return domain.User{}, fmt.Errorf("unsupported locale %q", locale)
	}
	row, err := a.queries.UpdateUserPreferredLocale(ctx, sqldb.UpdateUserPreferredLocaleParams{
		ID:              userID,
		PreferredLocale: locale,
	})
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (a *Auth) UpsertAuthIdentity(
	ctx context.Context,
	userID uuid.UUID,
	provider, providerSubject, email string,
	emailVerified bool,
) (domain.AuthIdentity, error) {
	row, err := a.queries.UpsertAuthIdentity(ctx, sqldb.UpsertAuthIdentityParams{
		UserID:          userID,
		Provider:        provider,
		ProviderSubject: providerSubject,
		Email:           email,
		EmailVerified:   emailVerified,
	})
	if err != nil {
		return domain.AuthIdentity{}, err
	}
	return mapAuthIdentity(row), nil
}

func (a *Auth) GetAuthIdentityByProviderSubject(
	ctx context.Context,
	provider, providerSubject string,
) (domain.AuthIdentity, error) {
	row, err := a.queries.GetAuthIdentityByProviderSubject(ctx, sqldb.GetAuthIdentityByProviderSubjectParams{
		Provider:        provider,
		ProviderSubject: providerSubject,
	})
	if err != nil {
		return domain.AuthIdentity{}, err
	}
	return mapAuthIdentity(row), nil
}

func (a *Auth) GetAuthIdentitiesByUser(ctx context.Context, userID uuid.UUID) ([]domain.AuthIdentity, error) {
	rows, err := a.queries.GetAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuthIdentity, len(rows))
	for i, row := range rows {
		out[i] = mapAuthIdentity(row)
	}
	return out, nil
}

func (a *Auth) CreateHousehold(ctx context.Context, name string) (domain.Household, error) {
	row, err := a.queries.CreateHousehold(ctx, name)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) GetHousehold(ctx context.Context, id uuid.UUID) (domain.Household, error) {
	row, err := a.queries.GetHousehold(ctx, id)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) ClaimHousehold(ctx context.Context, id uuid.UUID) (domain.Household, error) {
	row, err := a.queries.ClaimHousehold(ctx, id)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) GetUnclaimedSeedHousehold(ctx context.Context) (domain.Household, error) {
	row, err := a.queries.GetUnclaimedSeedHousehold(ctx)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) GetHouseholdByName(ctx context.Context, name string) (domain.Household, error) {
	row, err := a.queries.GetHouseholdByName(ctx, name)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) GetFirstHousehold(ctx context.Context) (domain.Household, error) {
	row, err := a.queries.GetFirstHousehold(ctx)
	if err != nil {
		return domain.Household{}, err
	}
	return mapHousehold(row), nil
}

func (a *Auth) CreateMembership(
	ctx context.Context,
	householdID, userID uuid.UUID,
	role string,
) (domain.HouseholdMembership, error) {
	row, err := a.queries.CreateMembership(ctx, sqldb.CreateMembershipParams{
		HouseholdID: householdID,
		UserID:      userID,
		Role:        role,
	})
	if err != nil {
		return domain.HouseholdMembership{}, err
	}
	return mapMembership(row), nil
}

func (a *Auth) GetMembershipsByUser(ctx context.Context, userID uuid.UUID) ([]domain.HouseholdMembership, error) {
	rows, err := a.queries.GetMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.HouseholdMembership, len(rows))
	for i, row := range rows {
		out[i] = mapMembership(row)
	}
	return out, nil
}

type CreateSessionInput struct {
	UserID            uuid.UUID
	TokenHash         []byte
	ExpiresAt         time.Time
	AbsoluteExpiresAt time.Time
	UserAgent         string
	IP                *netip.Addr
}

func (a *Auth) CreateSession(ctx context.Context, in CreateSessionInput) (domain.Session, error) {
	row, err := a.queries.CreateSession(ctx, sqldb.CreateSessionParams{
		UserID:            in.UserID,
		TokenHash:         in.TokenHash,
		ExpiresAt:         timestamptz(in.ExpiresAt),
		AbsoluteExpiresAt: timestamptz(in.AbsoluteExpiresAt),
		UserAgent:         in.UserAgent,
		Ip:                in.IP,
	})
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (a *Auth) GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (domain.Session, error) {
	row, err := a.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (a *Auth) TouchSession(ctx context.Context, id uuid.UUID, expiresAt time.Time) (domain.Session, error) {
	row, err := a.queries.TouchSession(ctx, sqldb.TouchSessionParams{
		ID:        id,
		ExpiresAt: timestamptz(expiresAt),
	})
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (a *Auth) RevokeSession(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	row, err := a.queries.RevokeSession(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (a *Auth) DeleteExpiredSessions(ctx context.Context) error {
	return a.queries.DeleteExpiredSessions(ctx)
}

type OAuthLoginState struct {
	State        string
	Nonce        string
	CodeVerifier string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

func (a *Auth) CreateOAuthLoginState(
	ctx context.Context,
	state, nonce, codeVerifier string,
	expiresAt time.Time,
) (OAuthLoginState, error) {
	row, err := a.queries.CreateOAuthLoginState(ctx, sqldb.CreateOAuthLoginStateParams{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ExpiresAt:    timestamptz(expiresAt),
	})
	if err != nil {
		return OAuthLoginState{}, err
	}
	return mapOAuthLoginState(row), nil
}

func (a *Auth) GetOAuthLoginState(ctx context.Context, state string) (OAuthLoginState, error) {
	row, err := a.queries.GetOAuthLoginState(ctx, state)
	if err != nil {
		return OAuthLoginState{}, err
	}
	return mapOAuthLoginState(row), nil
}

func (a *Auth) DeleteOAuthLoginState(ctx context.Context, state string) error {
	return a.queries.DeleteOAuthLoginState(ctx, state)
}

func (a *Auth) DeleteExpiredOAuthLoginStates(ctx context.Context) error {
	return a.queries.DeleteExpiredOAuthLoginStates(ctx)
}

func mapOAuthLoginState(row sqldb.OauthLoginState) OAuthLoginState {
	return OAuthLoginState{
		State:        row.State,
		Nonce:        row.Nonce,
		CodeVerifier: row.CodeVerifier,
		ExpiresAt:    row.ExpiresAt.Time,
		CreatedAt:    row.CreatedAt.Time,
	}
}

func mapUser(row sqldb.User) domain.User {
	return domain.User{
		ID:              row.ID,
		DisplayName:     row.DisplayName,
		GivenName:       textValue(row.GivenName),
		PictureURL:      textValue(row.PictureUrl),
		PreferredLocale: row.PreferredLocale,
		CreatedAt:       row.CreatedAt.Time,
	}
}

func textOrNull(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func mapAuthIdentity(row sqldb.AuthIdentity) domain.AuthIdentity {
	return domain.AuthIdentity{
		ID:              row.ID,
		UserID:          row.UserID,
		Provider:        row.Provider,
		ProviderSubject: row.ProviderSubject,
		Email:           row.Email,
		EmailVerified:   row.EmailVerified,
		CreatedAt:       row.CreatedAt.Time,
	}
}

func mapHousehold(row sqldb.Household) domain.Household {
	var claimedAt *time.Time
	if row.ClaimedAt.Valid {
		t := row.ClaimedAt.Time
		claimedAt = &t
	}
	return domain.Household{
		ID:        row.ID,
		Name:      row.Name,
		ClaimedAt: claimedAt,
		CreatedAt: row.CreatedAt.Time,
	}
}

func mapMembership(row sqldb.HouseholdMembership) domain.HouseholdMembership {
	return domain.HouseholdMembership{
		HouseholdID: row.HouseholdID,
		UserID:      row.UserID,
		Role:        row.Role,
		CreatedAt:   row.CreatedAt.Time,
	}
}

func mapSession(row sqldb.Session) domain.Session {
	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		t := row.RevokedAt.Time
		revokedAt = &t
	}
	return domain.Session{
		ID:                row.ID,
		UserID:            row.UserID,
		TokenHash:         append([]byte(nil), row.TokenHash...),
		ExpiresAt:         row.ExpiresAt.Time,
		AbsoluteExpiresAt: row.AbsoluteExpiresAt.Time,
		CreatedAt:         row.CreatedAt.Time,
		RevokedAt:         revokedAt,
		UserAgent:         row.UserAgent,
		IP:                row.Ip,
	}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}
