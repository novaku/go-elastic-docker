package service

import (
	"context"
	"errors"
	"testing"
)

type mockPinger struct {
	err error
}

func (m mockPinger) Ping() error { return m.err }

func TestElasticsearchHealthChecker_Check(t *testing.T) {
	t.Parallel()

	t.Run("healthy", func(t *testing.T) {
		t.Parallel()
		h := NewElasticsearchHealthChecker(mockPinger{})
		if err := h.Check(context.Background()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		t.Parallel()
		h := NewElasticsearchHealthChecker(mockPinger{err: errors.New("unreachable")})
		if err := h.Check(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}
