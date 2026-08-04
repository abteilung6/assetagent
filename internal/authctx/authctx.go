package authctx

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	KeyHouseholdID ctxKey = 1
	KeyUserID      ctxKey = 2
	KeySessionID   ctxKey = 3
)

func WithHouseholdID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, KeyHouseholdID, id)
}

func HouseholdID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(KeyHouseholdID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, KeyUserID, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(KeyUserID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func WithSessionID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, KeySessionID, id)
}

func SessionID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(KeySessionID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
