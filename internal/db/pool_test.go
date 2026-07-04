package db

import (
	"context"
	"testing"
)

func TestNewPool_invalidURL(t *testing.T) {
	ctx := context.Background()

	_, err := NewPool(ctx, "not-a-url")
	if err == nil {
		t.Fatal("NewPool() error = nil, want error")
	}
}
