package lot_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/helper"
)

func buildChain(t *testing.T) (uuid.UUID, []*lot.Event) {
	t.Helper()

	params := validStageEventParams()
	first, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := []*lot.Event{first}
	for range 2 {
		next, err := lot.NewTransferEvent(lot.TransferEventParams{
			LotID:         params.LotID,
			Previous:      events[len(events)-1],
			Stage:         lot.StageFarm,
			EventDate:     helper.DateUTC(time.Now().AddDate(0, 0, -1)),
			ActorID:       params.ActorID,
			ActorRole:     "producer",
			TargetActorID: uuid.New(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, next)
	}
	return params.LotID, events
}

func TestVerifyChainAcceptsAnIntactChain(t *testing.T) {
	lotID, events := buildChain(t)

	if !lot.VerifyChain(lotID, events) {
		t.Error("expected the chain to be valid")
	}
}

func TestVerifyChainAcceptsALotWithoutEvents(t *testing.T) {
	if !lot.VerifyChain(uuid.New(), nil) {
		t.Error("expected an empty chain to be valid")
	}
}

func TestVerifyChainDetectsBrokenChains(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(lotID uuid.UUID, events []*lot.Event) (uuid.UUID, []*lot.Event)
	}{
		{"tampered event", func(id uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			p := restoreParamsOf(e[1])
			p.Note = "Edited afterwards"
			e[1] = lot.RestoreEvent(p)
			return id, e
		}},
		{"missing first event", func(id uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			return id, e[1:]
		}},
		{"missing middle event", func(id uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			return id, []*lot.Event{e[0], e[2]}
		}},
		{"reordered events", func(id uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			e[1], e[2] = e[2], e[1]
			return id, e
		}},
		{"event of another lot", func(_ uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			return uuid.New(), e
		}},
		{"default event", func(id uuid.UUID, e []*lot.Event) (uuid.UUID, []*lot.Event) {
			e[1] = lot.DefaultEvent()
			return id, e
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lotID, events := tt.mutate(buildChain(t))

			if lot.VerifyChain(lotID, events) {
				t.Error("expected the chain to be invalid")
			}
		})
	}
}
