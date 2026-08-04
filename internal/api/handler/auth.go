package handler

import (
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/authctx"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		writeUnauthorizedError(w, "not authenticated")
		return
	}
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		writeUnauthorizedError(w, "not authenticated")
		return
	}

	user, household, membership, email, err := h.sessions.LoadMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeUnauthorizedError(w, "not authenticated")
			return
		}
		writeInternalError(w, "failed to load current user")
		return
	}

	resp := gen.MeResponse{
		User: gen.MeUser{
			Id:          user.ID,
			DisplayName: user.DisplayName,
		},
		Household: gen.MeHousehold{
			Id:   household.ID,
			Name: household.Name,
		},
		Membership: gen.MeMembership{
			Role: toMeMembershipRole(membership.Role),
		},
	}
	if email != "" {
		e := openapi_types.Email(email)
		resp.User.Email = &e
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) PostLogout(w http.ResponseWriter, r *http.Request) {
	cfg := service.SessionConfig{CookieName: "session"}
	if h.sessions != nil {
		cfg = h.sessions.Config()
		if sessionID, ok := authctx.SessionID(r.Context()); ok {
			if err := h.sessions.Logout(r.Context(), sessionID); err != nil {
				writeInternalError(w, "failed to revoke session")
				return
			}
		}
	}
	service.ClearSessionCookie(w, cfg)
	w.WriteHeader(http.StatusNoContent)
}

func toMeMembershipRole(role string) gen.MeMembershipRole {
	switch role {
	case domain.MembershipRoleMember:
		return gen.Member
	default:
		return gen.Owner
	}
}
