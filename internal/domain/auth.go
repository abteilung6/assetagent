package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

const (
	MembershipRoleOwner  = "owner"
	MembershipRoleMember = "member"

	AuthProviderGoogle = "google"

	SeedHouseholdName = "Local seed"
)

type User struct {
	ID          uuid.UUID
	DisplayName string
	CreatedAt   time.Time
}

type AuthIdentity struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Provider        string
	ProviderSubject string
	Email           string
	EmailVerified   bool
	CreatedAt       time.Time
}

type Household struct {
	ID        uuid.UUID
	Name      string
	ClaimedAt *time.Time
	CreatedAt time.Time
}

type HouseholdMembership struct {
	HouseholdID uuid.UUID
	UserID      uuid.UUID
	Role        string
	CreatedAt   time.Time
}

type Session struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	TokenHash          []byte
	ExpiresAt          time.Time
	AbsoluteExpiresAt  time.Time
	CreatedAt          time.Time
	RevokedAt          *time.Time
	UserAgent          string
	IP                 *netip.Addr
}
