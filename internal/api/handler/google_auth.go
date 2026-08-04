package handler

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/service"
)

// GoogleAuthHandler serves non-OpenAPI Google OIDC redirect endpoints.
type GoogleAuthHandler struct {
	google   *service.GoogleAuthService
	sessions *service.SessionService
}

func NewGoogleAuthHandler(google *service.GoogleAuthService, sessions *service.SessionService) *GoogleAuthHandler {
	return &GoogleAuthHandler{google: google, sessions: sessions}
}

func (h *GoogleAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	if h.google == nil || !h.google.Configured() {
		writeJSON(w, http.StatusServiceUnavailable, gen.Error{
			Error:   "google_auth_unavailable",
			Message: "Google sign-in is not configured",
		})
		return
	}

	authURL, err := h.google.Start(r.Context())
	if err != nil {
		if errors.Is(err, service.ErrGoogleAuthNotConfigured) {
			writeJSON(w, http.StatusServiceUnavailable, gen.Error{
				Error:   "google_auth_unavailable",
				Message: "Google sign-in is not configured",
			})
			return
		}
		writeInternalError(w, "failed to start Google sign-in")
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *GoogleAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil || !h.google.Configured() {
		writeJSON(w, http.StatusServiceUnavailable, gen.Error{
			Error:   "google_auth_unavailable",
			Message: "Google sign-in is not configured",
		})
		return
	}

	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		http.Redirect(w, r, h.google.FrontendURL()+"/login?error=oauth", http.StatusFound)
		return
	}

	state := q.Get("state")
	code := q.Get("code")
	ip := clientIP(r)
	rawToken, err := h.google.Complete(r.Context(), state, code, r.UserAgent(), ip)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailNotVerified):
			http.Redirect(w, r, h.google.FrontendURL()+"/login?error=email_unverified", http.StatusFound)
		case errors.Is(err, service.ErrOAuthStateInvalid),
			errors.Is(err, service.ErrOAuthExchangeFailed),
			errors.Is(err, service.ErrGoogleAuthNotConfigured):
			http.Redirect(w, r, h.google.FrontendURL()+"/login?error=oauth", http.StatusFound)
		default:
			http.Redirect(w, r, h.google.FrontendURL()+"/login?error=oauth", http.StatusFound)
		}
		return
	}

	cfg := service.SessionConfig{CookieName: "session"}
	if h.sessions != nil {
		cfg = h.sessions.Config()
	}
	service.SetSessionCookie(w, rawToken, cfg)
	http.Redirect(w, r, h.google.FrontendURL()+"/", http.StatusFound)
}

func clientIP(r *http.Request) *netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	return &addr
}
