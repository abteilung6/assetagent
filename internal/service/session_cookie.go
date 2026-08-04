package service

import (
	"net/http"
	"strings"
	"time"
)

// ResolveSessionCookieOptions applies production cookie hardening.
// In production the cookie is always __Host-session with Secure=true.
func ResolveSessionCookieOptions(appEnv, cookieName string, secure bool) (name string, secureOut bool) {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return "__Host-session", true
	}
	if cookieName == "" {
		cookieName = "session"
	}
	return cookieName, secure
}

// SetSessionCookie writes the opaque session token as an HttpOnly cookie.
func SetSessionCookie(w http.ResponseWriter, rawToken string, cfg SessionConfig) {
	maxAge := int(cfg.Absolute.Seconds())
	if maxAge <= 0 {
		maxAge = int(cfg.Idle.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		MaxAge:   maxAge,
	})
}

// ClearSessionCookie expires the session cookie.
func ClearSessionCookie(w http.ResponseWriter, cfg SessionConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		MaxAge:   -1,
	})
}

// SessionConfigFromEnv builds SessionConfig with production cookie overrides.
func SessionConfigFromEnv(appEnv, cookieName string, secure bool, idleHours, absoluteHours int) SessionConfig {
	name, sec := ResolveSessionCookieOptions(appEnv, cookieName, secure)
	idle := time.Duration(idleHours) * time.Hour
	absolute := time.Duration(absoluteHours) * time.Hour
	return SessionConfig{
		CookieName: name,
		Secure:     sec,
		Idle:       idle,
		Absolute:   absolute,
	}
}
