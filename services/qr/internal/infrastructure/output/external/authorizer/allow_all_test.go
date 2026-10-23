package authorizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/infrastructure/output/external/authorizer"
)

func TestAllowAllAlwaysAllows(t *testing.T) {
	allowed, err := authorizer.NewAllowAll().IsAllowed(
		context.Background(),
		ports.Subject{ActorID: uuid.New()},
		ports.ActionReadQRImage,
		ports.Resource{Type: ports.ResourceLot, ID: uuid.New()},
	)

	if err != nil || !allowed {
		t.Errorf("expected allowed without error, got %v %v", allowed, err)
	}
}
