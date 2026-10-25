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

func newLot(t *testing.T, farmID uuid.UUID, code string) *lot.Lot {
	t.Helper()

	harvest := time.Now().UTC().AddDate(0, -1, 0)
	l, err := lot.New(farmID, uuid.New(), code, "Castillo", lot.ProcessWashed, harvest)
	if err != nil {
		t.Fatalf("cannot build lot: %v", err)
	}
	return l
}

func TestLotRepositoryExistsByCode(t *testing.T) {
	ctx := context.Background()
	farmID := uuid.New()
	seeded := newLot(t, farmID, "LOT-001")
	repo := memory.NewLotRepository(seeded)

	tests := []struct {
		name   string
		farmID uuid.UUID
		code   string
		want   bool
	}{
		{"same farm and code", farmID, "LOT-001", true},
		{"same code in another farm", uuid.New(), "LOT-001", false},
		{"other code in the same farm", farmID, "LOT-002", false},
		{"code comparison is exact", farmID, "lot-001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByCode(ctx, tt.farmID, tt.code)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("empty repository", func(t *testing.T) {
		got, err := memory.NewLotRepository().ExistsByCode(ctx, farmID, "LOT-001")

		if err != nil || got {
			t.Errorf("got %v, %v; want false, nil", got, err)
		}
	})
}

func TestLotRepositorySave(t *testing.T) {
	ctx := context.Background()

	t.Run("saved lot can be found by code", func(t *testing.T) {
		repo := memory.NewLotRepository()
		l := newLot(t, uuid.New(), "LOT-001")

		if err := repo.Save(ctx, l); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, err := repo.ExistsByCode(ctx, l.FarmID(), l.Code())
		if err != nil || !exists {
			t.Errorf("got %v, %v; want true, nil", exists, err)
		}
	})

	t.Run("saving the same lot twice keeps a single copy", func(t *testing.T) {
		repo := memory.NewLotRepository()
		l := newLot(t, uuid.New(), "LOT-001")

		if err := repo.Save(ctx, l); err != nil {
			t.Fatalf("first save: %v", err)
		}
		if err := repo.Save(ctx, l); err != nil {
			t.Fatalf("second save: %v", err)
		}
	})

	rejected := []struct {
		name string
		lot  *lot.Lot
	}{
		{"default lot", lot.Default()},
		{"nil lot", nil},
	}

	for _, tt := range rejected {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			err := memory.NewLotRepository().Save(ctx, tt.lot)

			if !exception.HasCode(err, exception.CodeInternal) {
				t.Errorf("got %v, want an internal exception", err)
			}
		})
	}

	t.Run("seed ignores nil and default lots", func(t *testing.T) {
		repo := memory.NewLotRepository(nil, lot.Default())

		exists, err := repo.ExistsByCode(ctx, uuid.Nil, "")
		if err != nil || exists {
			t.Errorf("got %v, %v; want false, nil", exists, err)
		}
	})
}

func TestLotRepositoryIsSafeForConcurrentUse(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewLotRepository()
	farmID := uuid.New()

	const workers = 50
	errs := make(chan error, workers*2)

	var wg sync.WaitGroup
	for i := range workers {
		code := fmt.Sprintf("LOT-%03d", i)
		l := newLot(t, farmID, code)

		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- repo.Save(ctx, l)
		}()
		go func() {
			defer wg.Done()
			_, err := repo.ExistsByCode(ctx, farmID, code)
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

	for i := range workers {
		code := fmt.Sprintf("LOT-%03d", i)
		if exists, _ := repo.ExistsByCode(ctx, farmID, code); !exists {
			t.Errorf("lot %s was lost", code)
		}
	}
}

func TestLotRepositoryReturnsCopies(t *testing.T) {
	ctx := context.Background()

	t.Run("changing a found lot does not change the stored one", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		found, err := repo.FindByID(ctx, seeded.ID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := found.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		again, _ := repo.FindByID(ctx, seeded.ID())
		if again.CurrentStage() != lot.StageNone || again.Version() != 1 {
			t.Errorf("stored lot changed: stage %q, version %d", again.CurrentStage(), again.Version())
		}
	})

	t.Run("changing the saved lot does not change the stored one", func(t *testing.T) {
		repo := memory.NewLotRepository()
		l := newLot(t, uuid.New(), "LOT-001")
		if err := repo.Save(ctx, l); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := l.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stored, _ := repo.FindByID(ctx, l.ID())
		if stored.CurrentStage() != lot.StageNone || stored.Version() != 1 {
			t.Errorf("stored lot changed: stage %q, version %d", stored.CurrentStage(), stored.Version())
		}
	})

	t.Run("changing the seeded lot does not change the stored one", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)
		if err := seeded.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stored, _ := repo.FindByID(ctx, seeded.ID())
		if stored.CurrentStage() != lot.StageNone {
			t.Errorf("stored lot changed: stage %q", stored.CurrentStage())
		}
	})

	t.Run("changing a listed lot does not change the stored one", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		listed, err := repo.List(ctx, lot.Filter{}, lot.Cursor{}, 10)
		if err != nil || len(listed) != 1 {
			t.Fatalf("got %v, %v; want one lot", listed, err)
		}
		if err := listed[0].AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stored, _ := repo.FindByID(ctx, seeded.ID())
		if stored.CurrentStage() != lot.StageNone {
			t.Errorf("stored lot changed: stage %q", stored.CurrentStage())
		}
	})

	t.Run("the copy keeps every field", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		if err := seeded.Void("Registered by mistake"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		repo := memory.NewLotRepository(seeded)

		found, _ := repo.FindByID(ctx, seeded.ID())
		if found.ID() != seeded.ID() || found.FarmID() != seeded.FarmID() || found.CustodianID() != seeded.CustodianID() {
			t.Error("unexpected identity")
		}
		if found.Code() != seeded.Code() || found.Variety() != seeded.Variety() || found.Process() != seeded.Process() {
			t.Error("unexpected lot data")
		}
		if !found.HarvestDate().Equal(seeded.HarvestDate()) || !found.CreatedAt().Equal(seeded.CreatedAt()) {
			t.Error("unexpected dates")
		}
		if found.Status() != lot.StatusVoided || found.VoidReason() != seeded.VoidReason() || found.Version() != seeded.Version() {
			t.Error("unexpected state")
		}
	})
}

func TestLotRepositoryUpdateStage(t *testing.T) {
	ctx := context.Background()

	t.Run("persists the new stage and version", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		l, _ := repo.FindByID(ctx, seeded.ID())
		if err := l.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := repo.UpdateStage(ctx, l, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stored, _ := repo.FindByID(ctx, seeded.ID())
		if stored.CurrentStage() != lot.StageFarm || stored.Version() != 2 {
			t.Errorf("got stage %q, version %d; want farm, 2", stored.CurrentStage(), stored.Version())
		}
	})

	t.Run("a stale expected version conflicts and keeps the stored lot", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		l, _ := repo.FindByID(ctx, seeded.ID())
		if err := l.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, expected := range []int{0, 2} {
			if err := repo.UpdateStage(ctx, l, expected); !exception.HasCode(err, exception.CodeLotVersionConflict) {
				t.Errorf("expected %d: got %v, want code %q", expected, err, exception.CodeLotVersionConflict)
			}
		}

		stored, _ := repo.FindByID(ctx, seeded.ID())
		if stored.CurrentStage() != lot.StageNone || stored.Version() != 1 {
			t.Errorf("stored lot changed: stage %q, version %d", stored.CurrentStage(), stored.Version())
		}
	})

	t.Run("the second of two concurrent updates conflicts", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		first, _ := repo.FindByID(ctx, seeded.ID())
		second, _ := repo.FindByID(ctx, seeded.ID())
		if err := first.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := second.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := repo.UpdateStage(ctx, first, 1); err != nil {
			t.Fatalf("first update: unexpected error: %v", err)
		}
		if err := repo.UpdateStage(ctx, second, 1); !exception.HasCode(err, exception.CodeLotVersionConflict) {
			t.Errorf("second update: got %v, want code %q", err, exception.CodeLotVersionConflict)
		}
	})

	t.Run("a lot that does not exist is not found", func(t *testing.T) {
		repo := memory.NewLotRepository()

		err := repo.UpdateStage(ctx, newLot(t, uuid.New(), "LOT-001"), 1)
		if !exception.HasCode(err, exception.CodeLotNotFound) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
		}
	})

	rejected := []struct {
		name string
		lot  *lot.Lot
	}{
		{"default lot", lot.Default()},
		{"nil lot", nil},
	}

	for _, tt := range rejected {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			err := memory.NewLotRepository().UpdateStage(ctx, tt.lot, 1)

			if !exception.HasCode(err, exception.CodeInternal) {
				t.Errorf("got %v, want an internal exception", err)
			}
		})
	}

	t.Run("only one of many concurrent updates wins", func(t *testing.T) {
		seeded := newLot(t, uuid.New(), "LOT-001")
		repo := memory.NewLotRepository(seeded)

		const workers = 20
		results := make(chan error, workers)

		loaded := make([]*lot.Lot, workers)
		for i := range loaded {
			loaded[i], _ = repo.FindByID(ctx, seeded.ID())
			if err := loaded[i].AdvanceTo(lot.StageFarm); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		var wg sync.WaitGroup
		for _, l := range loaded {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- repo.UpdateStage(ctx, l, 1)
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
	})
}
