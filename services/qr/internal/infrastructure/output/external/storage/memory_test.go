package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMemoryPresignViewBuildsURLAndExpiry(t *testing.T) {
	before := time.Now().UTC()

	view, err := NewMemory().PresignView(context.Background(), "lot/ev/imagen.jpg", time.Hour)

	if err != nil || !strings.HasSuffix(view.URL, "/evidences/lot/ev/imagen.jpg") {
		t.Fatalf("got %+v %v", view, err)
	}
	if !view.ExpiresAt.After(before.Add(59 * time.Minute)) {
		t.Errorf("expiry too early: %v", view.ExpiresAt)
	}
}
