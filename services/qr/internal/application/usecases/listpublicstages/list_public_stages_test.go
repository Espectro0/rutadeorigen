package listpublicstages_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/listpublicstages"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeStages struct {
	items []*stageview.StageView
	err   error
	asked []uuid.UUID
}

func (f *fakeStages) Save(_ context.Context, _ *stageview.StageView) error { return nil }

func (f *fakeStages) ListByLotID(_ context.Context, lotID uuid.UUID) ([]*stageview.StageView, error) {
	f.asked = append(f.asked, lotID)
	if f.err != nil {
		return []*stageview.StageView{}, f.err
	}
	return f.items, nil
}

func lotViewWithStatus(t *testing.T, status lotview.Status) *lotview.LotView {
	t.Helper()

	view, err := lotview.New(lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Now().AddDate(0, -2, 0),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageRoasting,
		Status:       status,
		Version:      3,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return view
}

func storedStage(t *testing.T, lotID uuid.UUID, stage lotview.Stage) *stageview.StageView {
	t.Helper()

	view, err := stageview.New(stageview.NewParams{
		LotID:     lotID,
		Stage:     stage,
		EventDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
		Details:   stageview.Details{RoastLevel: "medium", RoastDurationMinutes: 12},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return view
}

func TestListPublicStages(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the stored stages of an active lot", func(t *testing.T) {
		view := lotViewWithStatus(t, lotview.StatusActive)
		stored := []*stageview.StageView{
			storedStage(t, view.LotID(), lotview.StageFarm),
			storedStage(t, view.LotID(), lotview.StageRoasting),
		}
		stages := &fakeStages{items: stored}

		got, err := listpublicstages.New(stages).Execute(ctx, view)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0] != stored[0] || got[1] != stored[1] {
			t.Errorf("unexpected stages: %+v", got)
		}
		if len(stages.asked) != 1 || stages.asked[0] != view.LotID() {
			t.Errorf("expected one query for the lot, got %v", stages.asked)
		}
	})

	t.Run("returns no stages for a voided lot without querying", func(t *testing.T) {
		view := lotViewWithStatus(t, lotview.StatusVoided)
		stages := &fakeStages{items: []*stageview.StageView{storedStage(t, view.LotID(), lotview.StageFarm)}}

		got, err := listpublicstages.New(stages).Execute(ctx, view)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 || len(stages.asked) != 0 {
			t.Errorf("expected no stages and no query, got %d stages and %d queries", len(got), len(stages.asked))
		}
	})

	t.Run("returns no stages for a default lot view without querying", func(t *testing.T) {
		stages := &fakeStages{}

		got, err := listpublicstages.New(stages).Execute(ctx, lotview.Default())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 || len(stages.asked) != 0 {
			t.Errorf("expected no stages and no query, got %d stages and %d queries", len(got), len(stages.asked))
		}
	})

	t.Run("wraps a repository failure keeping the cause", func(t *testing.T) {
		view := lotViewWithStatus(t, lotview.StatusActive)
		stages := &fakeStages{err: errBackend}

		got, err := listpublicstages.New(stages).Execute(ctx, view)

		if !exception.HasCode(err, exception.CodeInternal) || !errors.Is(err, errBackend) {
			t.Errorf("expected an internal error wrapping the cause, got %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("expected an empty non-nil slice, got %v", got)
		}
	})
}
