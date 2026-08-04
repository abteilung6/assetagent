package middleware

import (
	"net/http"
	"strings"

	"github.com/abteilung6/assetagent/internal/authctx"
	"github.com/abteilung6/assetagent/internal/service"
)

// ResolveSessionMiddleware attaches user/session/household from a valid session
// cookie. Missing or invalid sessions leave the context without auth values.
func ResolveSessionMiddleware(sessions *service.SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if sessions != nil {
				if c, err := r.Cookie(sessions.CookieName()); err == nil && c.Value != "" {
					user, household, _, session, resolveErr := sessions.ResolveSession(ctx, c.Value)
					if resolveErr == nil {
						ctx = authctx.WithUserID(ctx, user.ID)
						ctx = authctx.WithSessionID(ctx, session.ID)
						ctx = authctx.WithHouseholdID(ctx, household.ID)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthMiddleware returns 401 JSON for /api/* routes that lack a UserID,
// except /api/health. Auth routes under /auth/* are always public.
func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if isPublicPath(path) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/api/") {
			if _, ok := authctx.UserID(r.Context()); !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"not authenticated"}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	if path == "/api/health" || path == "/health" {
		return true
	}
	return strings.HasPrefix(path, "/auth/")
}

// CORSMiddleware allows credentialed requests from exact origins in the CSV list.
func CORSMiddleware(allowedOriginsCSV string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, part := range strings.Split(allowedOriginsCSV, ",") {
		origin := strings.TrimSpace(part)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
