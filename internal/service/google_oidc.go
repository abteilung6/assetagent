package service

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

// RealGoogleOAuth implements GoogleCodeExchanger using golang.org/x/oauth2.
type RealGoogleOAuth struct {
	config *oauth2.Config
}

func NewRealGoogleOAuth(clientID, clientSecret, redirectURL string) *RealGoogleOAuth {
	return &RealGoogleOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
	}
}

func (g *RealGoogleOAuth) AuthCodeURL(state, nonce, codeChallenge string) string {
	return g.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.AccessTypeOnline,
	)
}

func (g *RealGoogleOAuth) Exchange(ctx context.Context, code, codeVerifier string) (string, error) {
	token, err := g.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return "", err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return "", fmt.Errorf("missing id_token in token response")
	}
	return raw, nil
}

// RealGoogleIDTokenVerifier verifies Google ID tokens via go-oidc.
type RealGoogleIDTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewRealGoogleIDTokenVerifier(ctx context.Context, clientID string) (*RealGoogleIDTokenVerifier, error) {
	provider, err := oidc.NewProvider(ctx, googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}
	return &RealGoogleIDTokenVerifier{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (v *RealGoogleIDTokenVerifier) Verify(ctx context.Context, rawIDToken, expectedNonce string) (GoogleIDTokenClaims, error) {
	idToken, err := v.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return GoogleIDTokenClaims{}, err
	}
	if idToken.Nonce != expectedNonce {
		return GoogleIDTokenClaims{}, fmt.Errorf("nonce mismatch")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return GoogleIDTokenClaims{}, err
	}
	return GoogleIDTokenClaims{
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		Nonce:         idToken.Nonce,
	}, nil
}
