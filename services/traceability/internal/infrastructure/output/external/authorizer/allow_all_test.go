package authorizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
)

func TestAllowAllAllowsEveryRequest(t *testing.T) {
	subjects := []ports.Subject{
		{},
		{ActorID: uuid.New(), Role: actor.TypeProducer},
		{ActorID: uuid.New(), Role: actor.TypeBrand},
	}

	for _, subject := range subjects {
		allowed, err := authorizer.NewAllowAll().IsAllowed(context.Background(), subject,
			ports.ActionCreateLot, ports.Resource{Type: ports.ResourceFarm, ID: uuid.New()})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("subject %+v must be allowed", subject)
		}
	}
}
