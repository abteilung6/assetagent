package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrSessionInvalid = ErrUnauthorized
)

type SessionConfig struct {
	CookieName string
	Secure     bool
	Idle       time.Duration
	Absolute   time.Duration
}

type SessionService struct {
	auth *repository.Auth
	cfg  SessionConfig
	now  func() time.Time
}

func NewSession(auth *repository.Auth, cfg SessionConfig) *SessionService {
	if cfg.CookieName == "" {
		cfg.CookieName = "session"
	}
	if cfg.Idle <= 0 {
		cfg.Idle = 336 * time.Hour
	}
	if cfg.Absolute <= 0 {
		cfg.Absolute = 720 * time.Hour
	}
	return &SessionService{
		auth: auth,
		cfg:  cfg,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *SessionService) CookieName() string {
	return s.cfg.CookieName
}

func (s *SessionService) Config() SessionConfig {
	return s.cfg
}

func (s *SessionService) IssueSession(
	ctx context.Context,
	userID uuid.UUID,
	userAgent string,
	ip *netip.Addr,
) (rawToken string, session domain.Session, err error) {
	raw, err := randomToken()
	if err != nil {
		return "", domain.Session{}, err
	}
	sum := sha256.Sum256([]byte(raw))
	now := s.now()
	session, err = s.auth.CreateSession(ctx, repository.CreateSessionInput{
		UserID:            userID,
		TokenHash:         sum[:],
		ExpiresAt:         now.Add(s.cfg.Idle),
		AbsoluteExpiresAt: now.Add(s.cfg.Absolute),
		UserAgent:         userAgent,
		IP:                ip,
	})
	if err != nil {
		return "", domain.Session{}, err
	}
	return raw, session, nil
}

func (s *SessionService) ResolveSession(
	ctx context.Context,
	rawToken string,
) (domain.User, domain.Household, domain.HouseholdMembership, domain.Session, error) {
	if rawToken == "" {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, ErrUnauthorized
	}

	sum := sha256.Sum256([]byte(rawToken))
	session, err := s.auth.GetSessionByTokenHash(ctx, sum[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, ErrUnauthorized
		}
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, err
	}

	now := s.now()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) || !session.AbsoluteExpiresAt.After(now) {
		_ = s.auth.DeleteExpiredSessions(ctx)
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, ErrUnauthorized
	}

	idleExpiry := now.Add(s.cfg.Idle)
	newExpiry := idleExpiry
	if session.AbsoluteExpiresAt.Before(newExpiry) {
		newExpiry = session.AbsoluteExpiresAt
	}
	if !newExpiry.Equal(session.ExpiresAt) {
		session, err = s.auth.TouchSession(ctx, session.ID, newExpiry)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, ErrUnauthorized
			}
			return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, err
		}
	}

	user, household, membership, err := s.loadUserHousehold(ctx, session.UserID)
	if err != nil {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, domain.Session{}, err
	}
	return user, household, membership, session, nil
}

func (s *SessionService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.auth.RevokeSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

// LoadMe returns user, household, membership and optional email for an authenticated user.
func (s *SessionService) LoadMe(
	ctx context.Context,
	userID uuid.UUID,
) (domain.User, domain.Household, domain.HouseholdMembership, string, error) {
	user, household, membership, err := s.loadUserHousehold(ctx, userID)
	if err != nil {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, "", err
	}
	identities, err := s.auth.GetAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, "", err
	}
	email := ""
	if len(identities) > 0 {
		email = identities[0].Email
	}
	return user, household, membership, email, nil
}

func (s *SessionService) loadUserHousehold(
	ctx context.Context,
	userID uuid.UUID,
) (domain.User, domain.Household, domain.HouseholdMembership, error) {
	user, err := s.auth.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, ErrUnauthorized
		}
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, err
	}

	memberships, err := s.auth.GetMembershipsByUser(ctx, userID)
	if err != nil {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, err
	}
	if len(memberships) == 0 {
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, ErrUnauthorized
	}
	membership := memberships[0]

	household, err := s.auth.GetHousehold(ctx, membership.HouseholdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, ErrUnauthorized
		}
		return domain.User{}, domain.Household{}, domain.HouseholdMembership{}, err
	}
	return user, household, membership, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
