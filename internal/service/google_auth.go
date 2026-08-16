package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrGoogleAuthNotConfigured = errors.New("google auth not configured")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrOAuthStateInvalid       = errors.New("invalid oauth state")
	ErrOAuthExchangeFailed     = errors.New("oauth exchange failed")
)

const oauthStateTTL = 10 * time.Minute

// GoogleIDTokenClaims are the verified ID token fields used for signup/login.
type GoogleIDTokenClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	GivenName     string
	PictureURL    string
	Locale        string
	Nonce         string
}

// GoogleCodeExchanger builds authorize URLs and exchanges auth codes for ID tokens.
type GoogleCodeExchanger interface {
	AuthCodeURL(state, nonce, codeChallenge string) string
	Exchange(ctx context.Context, code, codeVerifier string) (rawIDToken string, err error)
}

// GoogleIDTokenVerifier verifies a Google ID token and returns claims.
type GoogleIDTokenVerifier interface {
	Verify(ctx context.Context, rawIDToken, expectedNonce string) (GoogleIDTokenClaims, error)
}

type GoogleAuthConfig struct {
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	FrontendURL       string
	ClaimExistingData bool
	Configured        bool
}

type GoogleAuthService struct {
	auth      *repository.Auth
	sessions  *SessionService
	exchanger GoogleCodeExchanger
	verifier  GoogleIDTokenVerifier
	cfg       GoogleAuthConfig
	now       func() time.Time
}

func NewGoogleAuth(
	auth *repository.Auth,
	sessions *SessionService,
	exchanger GoogleCodeExchanger,
	verifier GoogleIDTokenVerifier,
	cfg GoogleAuthConfig,
) *GoogleAuthService {
	return &GoogleAuthService{
		auth:      auth,
		sessions:  sessions,
		exchanger: exchanger,
		verifier:  verifier,
		cfg:       cfg,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (g *GoogleAuthService) Configured() bool {
	return g != nil && g.cfg.Configured && g.exchanger != nil && g.verifier != nil
}

func (g *GoogleAuthService) FrontendURL() string {
	if g == nil || g.cfg.FrontendURL == "" {
		return "http://localhost:5173"
	}
	return strings.TrimRight(g.cfg.FrontendURL, "/")
}

// Start begins the Google OIDC authorization-code + PKCE flow.
func (g *GoogleAuthService) Start(ctx context.Context) (authURL string, err error) {
	if !g.Configured() {
		return "", ErrGoogleAuthNotConfigured
	}

	_ = g.auth.DeleteExpiredOAuthLoginStates(ctx)

	state, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	challenge := pkceS256Challenge(verifier)

	expiresAt := g.now().Add(oauthStateTTL)
	if _, err := g.auth.CreateOAuthLoginState(ctx, state, nonce, verifier, expiresAt); err != nil {
		return "", fmt.Errorf("store oauth state: %w", err)
	}

	return g.exchanger.AuthCodeURL(state, nonce, challenge), nil
}

// Complete exchanges the callback code, upserts identity/household, and issues a session.
func (g *GoogleAuthService) Complete(
	ctx context.Context,
	state, code, userAgent string,
	ip *netip.Addr,
) (rawToken string, err error) {
	if !g.Configured() {
		return "", ErrGoogleAuthNotConfigured
	}
	if state == "" || code == "" {
		return "", ErrOAuthStateInvalid
	}

	loginState, err := g.auth.GetOAuthLoginState(ctx, state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrOAuthStateInvalid
		}
		return "", err
	}
	_ = g.auth.DeleteOAuthLoginState(ctx, state)
	if !loginState.ExpiresAt.After(g.now()) {
		return "", ErrOAuthStateInvalid
	}

	rawIDToken, err := g.exchanger.Exchange(ctx, code, loginState.CodeVerifier)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}

	claims, err := g.verifier.Verify(ctx, rawIDToken, loginState.Nonce)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}
	if !claims.EmailVerified {
		return "", ErrEmailNotVerified
	}
	if claims.Subject == "" {
		return "", ErrOAuthExchangeFailed
	}

	userID, err := g.resolveOrCreateUserID(ctx, claims)
	if err != nil {
		return "", err
	}

	rawToken, _, err = g.sessions.IssueSession(ctx, userID, userAgent, ip)
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

func (g *GoogleAuthService) resolveOrCreateUserID(ctx context.Context, claims GoogleIDTokenClaims) (uuid.UUID, error) {
	identity, err := g.auth.GetAuthIdentityByProviderSubject(ctx, domain.AuthProviderGoogle, claims.Subject)
	if err == nil {
		if _, upsertErr := g.auth.UpsertAuthIdentity(
			ctx,
			identity.UserID,
			domain.AuthProviderGoogle,
			claims.Subject,
			claims.Email,
			claims.EmailVerified,
		); upsertErr != nil {
			return uuid.Nil, upsertErr
		}
		if _, profileErr := g.auth.UpdateUserGoogleProfile(
			ctx,
			identity.UserID,
			claims.GivenName,
			claims.PictureURL,
		); profileErr != nil {
			return uuid.Nil, profileErr
		}
		return identity.UserID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(claims.Email)
	}
	user, err := g.auth.CreateUser(ctx, repository.CreateUserInput{
		DisplayName:     displayName,
		GivenName:       claims.GivenName,
		PictureURL:      claims.PictureURL,
		PreferredLocale: domain.NormalizeLocale(claims.Locale),
	})
	if err != nil {
		return uuid.Nil, err
	}

	if _, err := g.auth.UpsertAuthIdentity(
		ctx,
		user.ID,
		domain.AuthProviderGoogle,
		claims.Subject,
		claims.Email,
		claims.EmailVerified,
	); err != nil {
		return uuid.Nil, err
	}

	if err := g.attachHousehold(ctx, user.ID, displayName); err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

func (g *GoogleAuthService) attachHousehold(ctx context.Context, userID uuid.UUID, displayName string) error {
	if g.cfg.ClaimExistingData {
		seed, err := g.auth.GetUnclaimedSeedHousehold(ctx)
		if err == nil {
			claimed, claimErr := g.auth.ClaimHousehold(ctx, seed.ID)
			if claimErr == nil {
				_, memErr := g.auth.CreateMembership(ctx, claimed.ID, userID, domain.MembershipRoleOwner)
				return memErr
			}
			if !errors.Is(claimErr, pgx.ErrNoRows) {
				return claimErr
			}
			// Lost the claim race — fall through to empty household.
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	name := "Household"
	if displayName != "" {
		name = displayName + "'s household"
	}
	household, err := g.auth.CreateHousehold(ctx, name)
	if err != nil {
		return err
	}
	_, err = g.auth.CreateMembership(ctx, household.ID, userID, domain.MembershipRoleOwner)
	return err
}

func pkceS256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
