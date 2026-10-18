package repotest

import (
	"bytes"
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Seeder func(t *testing.T, farmID, custodianID uuid.UUID)

func buildLot(farmID, custodianID uuid.UUID, code string, createdAt time.Time) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       farmID,
		CustodianID:  custodianID,
		Code:         code,
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageNone,
		Status:       lot.StatusActive,
		Version:      1,
		CreatedAt:    createdAt,
	})
}

func save(t *testing.T, repo ports.LotRepository, lots ...*lot.Lot) {
	t.Helper()

	for _, l := range lots {
		if err := repo.Save(context.Background(), l); err != nil {
			t.Fatalf("save %s: %v", l.Code(), err)
		}
	}
}

func ids(lots []*lot.Lot) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(lots))
	for _, l := range lots {
		out = append(out, l.ID())
	}
	return out
}

func newestFirst(lots []*lot.Lot) []*lot.Lot {
	sorted := append([]*lot.Lot(nil), lots...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].CreatedAt().Equal(sorted[j].CreatedAt()) {
			return sorted[i].CreatedAt().After(sorted[j].CreatedAt())
		}
		a, b := sorted[i].ID(), sorted[j].ID()
		return bytes.Compare(a[:], b[:]) > 0
	})
	return sorted
}

func RunLotRepositoryContract(t *testing.T, repo ports.LotRepository, seed Seeder) {
	ctx := context.Background()

	t.Run("FindByID returns the default lot when it does not exist", func(t *testing.T) {
		got, err := repo.FindByID(ctx, uuid.New())

		if err != nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want the default lot and no error", got, err)
		}
	})

	t.Run("FindByID returns the saved lot", func(t *testing.T) {
		farmID, custodianID := uuid.New(), uuid.New()
		seed(t, farmID, custodianID)
		saved := buildLot(farmID, custodianID, "LOT-001", time.Now().UTC().Truncate(time.Microsecond))
		save(t, repo, saved)

		got, err := repo.FindByID(ctx, saved.ID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID() != saved.ID() || got.FarmID() != farmID || got.CustodianID() != custodianID ||
			got.Code() != "LOT-001" || got.Version() != 1 || !got.CreatedAt().Equal(saved.CreatedAt()) {
			t.Errorf("got %+v, want the saved lot", got)
		}
	})

	t.Run("a code with SQL syntax is stored and read back as plain text", func(t *testing.T) {
		farmID, custodianID := uuid.New(), uuid.New()
		seed(t, farmID, custodianID)
		code := "LOT'); DROP TABLE lot;--"
		saved := buildLot(farmID, custodianID, code, time.Now().UTC().Truncate(time.Microsecond))
		save(t, repo, saved)

		got, err := repo.FindByID(ctx, saved.ID())

		if err != nil || got.Code() != code {
			t.Errorf("got %q, %v; want %q", got.Code(), err, code)
		}
		exists, err := repo.ExistsByCode(ctx, farmID, code)
		if err != nil || !exists {
			t.Errorf("ExistsByCode: got %v, %v; want true", exists, err)
		}
	})

	t.Run("List returns nothing for a farm without lots", func(t *testing.T) {
		got, err := repo.List(ctx, lot.Filter{FarmID: uuid.New()}, lot.Cursor{}, 10)

		if err != nil || len(got) != 0 {
			t.Errorf("got %d lots, %v; want none", len(got), err)
		}
	})

	t.Run("List pages newest first without gaps or repeats", func(t *testing.T) {
		farmID, custodianID := uuid.New(), uuid.New()
		seed(t, farmID, custodianID)
		base := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
		saved := []*lot.Lot{
			buildLot(farmID, custodianID, "LOT-001", base),
			buildLot(farmID, custodianID, "LOT-002", base.Add(1*time.Minute)),
			buildLot(farmID, custodianID, "LOT-003", base.Add(2*time.Minute)),
			buildLot(farmID, custodianID, "LOT-004", base.Add(3*time.Minute)),
			buildLot(farmID, custodianID, "LOT-005", base.Add(4*time.Minute)),
			buildLot(farmID, custodianID, "LOT-006", base.Add(4*time.Minute)),
		}
		save(t, repo, saved...)
		want := ids(newestFirst(saved))

		const pageSize = 2
		var got []uuid.UUID
		after := lot.Cursor{}
		for pages := 0; pages < len(saved)+1; pages++ {
			rows, err := repo.List(ctx, lot.Filter{FarmID: farmID}, after, pageSize)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			got = append(got, ids(rows)...)
			if len(rows) < pageSize {
				break
			}
			after = lot.CursorOf(rows[len(rows)-1])
		}

		if len(got) != len(want) {
			t.Fatalf("got %d lots, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("position %d: got %s, want %s", i, got[i], want[i])
			}
		}
	})

	t.Run("List respects the limit", func(t *testing.T) {
		farmID, custodianID := uuid.New(), uuid.New()
		seed(t, farmID, custodianID)
		base := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
		save(t, repo,
			buildLot(farmID, custodianID, "LOT-001", base),
			buildLot(farmID, custodianID, "LOT-002", base.Add(time.Minute)),
			buildLot(farmID, custodianID, "LOT-003", base.Add(2*time.Minute)),
		)

		got, err := repo.List(ctx, lot.Filter{FarmID: farmID}, lot.Cursor{}, 2)

		if err != nil || len(got) != 2 {
			t.Errorf("got %d lots, %v; want 2", len(got), err)
		}
	})

	t.Run("List filters by custodian", func(t *testing.T) {
		farmID, custodianA, custodianB := uuid.New(), uuid.New(), uuid.New()
		seed(t, farmID, custodianA)
		seed(t, uuid.New(), custodianB)
		now := time.Now().UTC().Truncate(time.Microsecond)
		mine := buildLot(farmID, custodianA, "LOT-001", now)
		other := buildLot(farmID, custodianB, "LOT-002", now.Add(time.Second))
		save(t, repo, mine, other)

		got, err := repo.List(ctx, lot.Filter{FarmID: farmID, CustodianID: custodianA}, lot.Cursor{}, 10)

		if err != nil || len(got) != 1 || got[0].ID() != mine.ID() {
			t.Errorf("got %v, %v; want only the lot of custodian A", ids(got), err)
		}
	})

	t.Run("List without a farm filter includes lots from every farm", func(t *testing.T) {
		custodianID := uuid.New()
		farmA, farmB := uuid.New(), uuid.New()
		seed(t, farmA, custodianID)
		seed(t, farmB, custodianID)
		now := time.Now().UTC().Truncate(time.Microsecond)
		inA := buildLot(farmA, custodianID, "LOT-001", now)
		inB := buildLot(farmB, custodianID, "LOT-001", now.Add(time.Second))
		save(t, repo, inA, inB)

		got, err := repo.List(ctx, lot.Filter{CustodianID: custodianID}, lot.Cursor{}, 10)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := ids(newestFirst([]*lot.Lot{inA, inB}))
		gotIDs := ids(got)
		if len(gotIDs) != 2 || gotIDs[0] != want[0] || gotIDs[1] != want[1] {
			t.Errorf("got %v, want %v", gotIDs, want)
		}
	})
}
