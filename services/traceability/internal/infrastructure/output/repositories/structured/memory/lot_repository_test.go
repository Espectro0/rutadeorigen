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
