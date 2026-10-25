package envelope_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/domain/actor"
)

type payload struct {
	Value string
}

func newActor(t *testing.T) *actor.Actor {
	t.Helper()
	a, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestNew(t *testing.T) {
	a := newActor(t)
	aggregateID := uuid.New()
	ctx := requestcontext.WithCorrelationID(context.Background(), "req-123")

	before := time.Now().UTC().Add(-time.Second)
	got := envelope.New(ctx, a, ports.EventLotCreated, ports.AggregateLot, aggregateID, 3, payload{Value: "x"})

	if got.ID == uuid.Nil {
		t.Error("expected a generated ID")
	}
	if got.Type != ports.EventLotCreated || got.AggregateType != ports.AggregateLot {
		t.Errorf("unexpected type %v / %v", got.Type, got.AggregateType)
	}
	if got.AggregateID != aggregateID || got.Version != 3 {
		t.Errorf("unexpected aggregate %s / version %d", got.AggregateID, got.Version)
	}
	if got.ActorID != a.ID() || got.ActorRole != actor.TypeProducer.String() {
		t.Error("unexpected actor")
	}
	if got.CorrelationID != "req-123" {
		t.Errorf("got correlation %q, want %q", got.CorrelationID, "req-123")
	}
	if got.OccurredAt.Before(before) || got.OccurredAt.After(time.Now().UTC().Add(time.Second)) {
		t.Errorf("unexpected occurrence time %v", got.OccurredAt)
	}
	if got.OccurredAt.Location() != time.UTC {
		t.Error("expected the occurrence time in UTC")
	}
	if p, ok := got.Payload.(payload); !ok || p.Value != "x" {
		t.Errorf("unexpected payload %v", got.Payload)
	}
}

func TestNewWithoutCorrelationID(t *testing.T) {
	got := envelope.New(context.Background(), newActor(t), ports.EventAccessDenied, ports.AggregateLot, uuid.Nil, 0, nil)

	if got.CorrelationID != "" {
		t.Errorf("got correlation %q, want it empty", got.CorrelationID)
	}
	if got.Payload != nil {
		t.Errorf("got payload %v, want nil", got.Payload)
	}
}

func TestNewGeneratesDistinctIDs(t *testing.T) {
	a := newActor(t)

	first := envelope.New(context.Background(), a, ports.EventLotCreated, ports.AggregateLot, uuid.Nil, 1, nil)
	second := envelope.New(context.Background(), a, ports.EventLotCreated, ports.AggregateLot, uuid.Nil, 1, nil)

	if first.ID == second.ID {
		t.Error("expected a distinct ID on every call")
	}
}
