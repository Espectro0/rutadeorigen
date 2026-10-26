package memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/platform/exception"
)

func newEvent(t *testing.T, lotID uuid.UUID, previous *lot.Event, stage lot.Stage, details lot.StageDetails) *lot.Event {
	t.Helper()

	e, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       lotID,
		HarvestDate: time.Now().AddDate(0, -1, 0),
		Previous:    previous,
		Stage:       stage,
		EventDate:   time.Now().AddDate(0, 0, -1),
		Note:        "Dried on raised beds",
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details:     details,
	})
	if err != nil {
		t.Fatalf("cannot build lot event: %v", err)
	}
	return e
}

func farmDetails() lot.FarmDetails {
	return lot.FarmDetails{
		DryingMethod:      lot.DryingMethodSunPatio,
		DryingDays:        15,
		MoisturePercent:   11.5,
		ParchmentWeightKg: 120.5,
	}
}

func roastingDetails() lot.RoastingDetails {
	return lot.RoastingDetails{
		RoastLevel:      lot.RoastLevelMedium,
		DurationMinutes: 14,
		GreenWeightKg:   100,
		RoastedWeightKg: 84,
	}
}

func newFarmEvent(t *testing.T, lotID uuid.UUID) *lot.Event {
	t.Helper()
	return newEvent(t, lotID, nil, lot.StageFarm, farmDetails())
}

func restoredEvent(lotID uuid.UUID, sequence int, hash string) *lot.Event {
	return lot.RestoreEvent(lot.RestoreEventParams{
		ID:           uuid.New(),
		LotID:        lotID,
		Sequence:     sequence,
		EventType:    lot.EventTypeStage,
		Stage:        lot.StageFarm,
		EventDate:    time.Now().UTC().AddDate(0, 0, -1),
		ActorID:      uuid.New(),
		ActorRole:    "producer",
		Details:      farmDetails(),
		PreviousHash: lot.GenesisHash,
		Hash:         hash,
		RegisteredAt: time.Now().UTC(),
	})
}

func TestLotEventRepositorySaveAndFindLast(t *testing.T) {
	ctx := context.Background()
	lotID := uuid.New()
	repo := memory.NewLotEventRepository()
	first := newFarmEvent(t, lotID)
	second := newEvent(t, lotID, first, lot.StageRoasting, roastingDetails())

	for _, e := range []*lot.Event{first, second} {
		if err := repo.Save(ctx, e); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	got, err := repo.FindLast(ctx, lotID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID() != second.ID() || got.Sequence() != 2 {
		t.Errorf("got event %s with sequence %d, want %s with sequence 2", got.ID(), got.Sequence(), second.ID())
	}
	if !got.VerifyHash() {
		t.Error("the stored event must keep a verifiable hash")
	}
	if got.Details() == nil || got.Details().Stage() != lot.StageRoasting {
		t.Error("the stored event must keep its details")
	}
}

func TestLotEventRepositoryFindLast(t *testing.T) {
	ctx := context.Background()

	t.Run("a lot without events returns the default event", func(t *testing.T) {
		got, err := memory.NewLotEventRepository().FindLast(ctx, uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.IsDefault() {
			t.Error("expected the default event")
		}
	})

	t.Run("returns the highest sequence even if it was saved first", func(t *testing.T) {
		lotID := uuid.New()
		repo := memory.NewLotEventRepository()
		for _, sequence := range []int{3, 1, 2} {
			if err := repo.Save(ctx, restoredEvent(lotID, sequence, fmt.Sprintf("hash-%d", sequence))); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		got, _ := repo.FindLast(ctx, lotID)
		if got.Sequence() != 3 {
			t.Errorf("got sequence %d, want 3", got.Sequence())
		}
	})

	t.Run("events of other lots are ignored", func(t *testing.T) {
		repo := memory.NewLotEventRepository()
		mine, other := uuid.New(), uuid.New()
		if err := repo.Save(ctx, newFarmEvent(t, other)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := repo.FindLast(ctx, mine)
		if !got.IsDefault() {
			t.Error("expected the default event for a lot without events")
		}
	})
}

func TestLotEventRepositoryListByLotID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the events ordered by sequence", func(t *testing.T) {
		lotID := uuid.New()
		repo := memory.NewLotEventRepository()
		for _, sequence := range []int{2, 3, 1} {
			if err := repo.Save(ctx, restoredEvent(lotID, sequence, fmt.Sprintf("hash-%d", sequence))); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		got, err := repo.ListByLotID(ctx, lotID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d events, want 3", len(got))
		}
		for i, e := range got {
			if e.Sequence() != i+1 {
				t.Errorf("position %d: got sequence %d, want %d", i, e.Sequence(), i+1)
			}
		}
	})

	t.Run("only returns the events of the lot", func(t *testing.T) {
		repo := memory.NewLotEventRepository()
		mine, other := uuid.New(), uuid.New()
		for _, id := range []uuid.UUID{mine, other} {
			if err := repo.Save(ctx, newFarmEvent(t, id)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		got, _ := repo.ListByLotID(ctx, mine)
		if len(got) != 1 || got[0].LotID() != mine {
			t.Errorf("got %d events, want only the one of the lot", len(got))
		}
	})

	t.Run("a lot without events returns an empty list", func(t *testing.T) {
		got, err := memory.NewLotEventRepository().ListByLotID(ctx, uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("got %v, want an empty non-nil list", got)
		}
	})

	t.Run("the chain of a full lot verifies after reloading", func(t *testing.T) {
		lotID := uuid.New()
		repo := memory.NewLotEventRepository()
		farm := newFarmEvent(t, lotID)
		roasting := newEvent(t, lotID, farm, lot.StageRoasting, roastingDetails())
		for _, e := range []*lot.Event{farm, roasting} {
			if err := repo.Save(ctx, e); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		got, _ := repo.ListByLotID(ctx, lotID)
		if got[0].PreviousHash() != lot.GenesisHash || got[1].PreviousHash() != got[0].Hash() {
			t.Error("the hash chain is broken")
		}
		for i, e := range got {
			if !e.VerifyHash() {
				t.Errorf("event %d does not verify", i+1)
			}
		}
	})
}

func TestLotEventRepositorySaveRejects(t *testing.T) {
	ctx := context.Background()

	t.Run("default and nil events", func(t *testing.T) {
		for _, e := range []*lot.Event{lot.DefaultEvent(), nil} {
			err := memory.NewLotEventRepository().Save(ctx, e)

			if !exception.HasCode(err, exception.CodeInternal) {
				t.Errorf("got %v, want an internal exception", err)
			}
		}
	})

	t.Run("a sequence lower than one", func(t *testing.T) {
		err := memory.NewLotEventRepository().Save(ctx, restoredEvent(uuid.New(), 0, "hash"))

		if !exception.HasCode(err, exception.CodeInternal) {
			t.Errorf("got %v, want an internal exception", err)
		}
	})

	t.Run("the same sequence in the same lot conflicts and keeps the first event", func(t *testing.T) {
		lotID := uuid.New()
		repo := memory.NewLotEventRepository()
		first := restoredEvent(lotID, 1, "hash-a")
		if err := repo.Save(ctx, first); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := repo.Save(ctx, restoredEvent(lotID, 1, "hash-b"))
		if !exception.HasCode(err, exception.CodeLotVersionConflict) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotVersionConflict)
		}

		got, _ := repo.ListByLotID(ctx, lotID)
		if len(got) != 1 || got[0].ID() != first.ID() {
			t.Error("the first event must be kept")
		}
	})

	t.Run("the same sequence in another lot is accepted", func(t *testing.T) {
		repo := memory.NewLotEventRepository()

		for _, hash := range []string{"hash-a", "hash-b"} {
			if err := repo.Save(ctx, restoredEvent(uuid.New(), 1, hash)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	})

	t.Run("a repeated event ID", func(t *testing.T) {
		repo := memory.NewLotEventRepository()
		e := newFarmEvent(t, uuid.New())
		if err := repo.Save(ctx, e); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := repo.Save(ctx, e); !exception.HasCode(err, exception.CodeInternal) {
			t.Errorf("got %v, want an internal exception", err)
		}
	})

	t.Run("a repeated hash", func(t *testing.T) {
		repo := memory.NewLotEventRepository()
		if err := repo.Save(ctx, restoredEvent(uuid.New(), 1, "same-hash")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := repo.Save(ctx, restoredEvent(uuid.New(), 1, "same-hash"))
		if !exception.HasCode(err, exception.CodeInternal) {
			t.Errorf("got %v, want an internal exception", err)
		}
	})
}

func TestLotEventRepositorySeed(t *testing.T) {
	ctx := context.Background()
	lotID := uuid.New()
	seeded := newFarmEvent(t, lotID)

	repo := memory.NewLotEventRepository(nil, lot.DefaultEvent(), seeded)

	got, _ := repo.ListByLotID(ctx, lotID)
	if len(got) != 1 || got[0].ID() != seeded.ID() {
		t.Errorf("got %d events, want only the seeded one", len(got))
	}
}

func TestLotEventRepositoryReturnsCopies(t *testing.T) {
	ctx := context.Background()
	lotID := uuid.New()
	repo := memory.NewLotEventRepository(newFarmEvent(t, lotID))

	t.Run("changing the returned list does not change the stored events", func(t *testing.T) {
		listed, _ := repo.ListByLotID(ctx, lotID)
		listed[0] = lot.DefaultEvent()

		again, _ := repo.ListByLotID(ctx, lotID)
		if len(again) != 1 || again[0].IsDefault() {
			t.Error("the stored events changed")
		}
	})

	t.Run("a returned event keeps every field", func(t *testing.T) {
		stored, _ := repo.ListByLotID(ctx, lotID)
		last, _ := repo.FindLast(ctx, lotID)

		if last == stored[0] {
			t.Error("expected independent copies")
		}
		if last.ID() != stored[0].ID() || last.Hash() != stored[0].Hash() || last.Sequence() != stored[0].Sequence() {
			t.Error("unexpected identity")
		}
		if last.Note() != "Dried on raised beds" || last.ActorRole() != "producer" || last.Type() != lot.EventTypeStage {
			t.Error("unexpected content")
		}
		if !last.EventDate().Equal(stored[0].EventDate()) || !last.RegisteredAt().Equal(stored[0].RegisteredAt()) {
			t.Error("unexpected dates")
		}
	})
}

func TestLotEventRepositoryIsSafeForConcurrentUse(t *testing.T) {
	ctx := context.Background()

	t.Run("saves and reads from many lots", func(t *testing.T) {
		repo := memory.NewLotEventRepository()

		const workers = 50
		errs := make(chan error, workers*3)
		lotIDs := make([]uuid.UUID, workers)

		var wg sync.WaitGroup
		for i := range workers {
			lotIDs[i] = uuid.New()
			e := newFarmEvent(t, lotIDs[i])
			id := lotIDs[i]

			wg.Add(3)
			go func() {
				defer wg.Done()
				errs <- repo.Save(ctx, e)
			}()
			go func() {
				defer wg.Done()
				_, err := repo.FindLast(ctx, id)
				errs <- err
			}()
			go func() {
				defer wg.Done()
				_, err := repo.ListByLotID(ctx, id)
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		for i, id := range lotIDs {
			if last, _ := repo.FindLast(ctx, id); last.IsDefault() {
				t.Errorf("the event of lot %d was lost", i)
			}
		}
	})

	t.Run("only one of many competing events for the same sequence wins", func(t *testing.T) {
		repo := memory.NewLotEventRepository()
		lotID := uuid.New()

		const workers = 20
		results := make(chan error, workers)

		var wg sync.WaitGroup
		for i := range workers {
			e := restoredEvent(lotID, 1, fmt.Sprintf("hash-%d", i))

			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- repo.Save(ctx, e)
			}()
		}
		wg.Wait()
		close(results)

		wins := 0
		for err := range results {
			switch {
			case err == nil:
				wins++
			case !exception.HasCode(err, exception.CodeLotVersionConflict):
				t.Errorf("unexpected error: %v", err)
			}
		}
		if wins != 1 {
			t.Errorf("got %d winners, want 1", wins)
		}

		stored, _ := repo.ListByLotID(ctx, lotID)
		if len(stored) != 1 {
			t.Errorf("got %d stored events, want 1", len(stored))
		}
	})
}
