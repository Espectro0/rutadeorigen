package getpubliclot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/getpubliclot"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeCache struct {
	item *lotview.LotView
	sets int
}

func (c *fakeCache) Get(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	if c.item == nil {
		return lotview.Default(), nil
	}
	return c.item, nil
}

func (c *fakeCache) Set(_ context.Context, _ *lotview.LotView) error {
	c.sets++
	return nil
}

func (c *fakeCache) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type fakeViews struct {
	view  *lotview.LotView
	err   error
	calls int
}

func (v *fakeViews) FindByID(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	v.calls++
	if v.err != nil {
		return lotview.Default(), v.err
	}
	if v.view == nil {
		return lotview.Default(), nil
	}
	return v.view, nil
}

func (v *fakeViews) Save(_ context.Context, _ *lotview.LotView) error { return nil }

func newView(t *testing.T, status lotview.Status) *lotview.LotView {
	t.Helper()

	v, err := lotview.New(lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Now().AddDate(0, -1, 0),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageRoasting,
		Status:       status,
		Version:      2,
	})
	if err != nil {
		t.Fatalf("cannot build lot view: %v", err)
	}
	return v
}

func assertCode(t *testing.T, err error, want exception.Code) {
	t.Helper()

	ex, ok := exception.From(err)
	if !ok {
		t.Fatalf("expected an *Exception, got %v", err)
	}
	if ex.Code() != want {
		t.Errorf("Code: got %v, want %v", ex.Code(), want)
	}
}

func TestGetPublicLotExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the lot from the repository and caches it", func(t *testing.T) {
		v := newView(t, lotview.StatusActive)
		cache, views := &fakeCache{}, &fakeViews{view: v}
		uc := getpubliclot.New(views, cache)

		got, err := uc.Execute(ctx, v.LotID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != v || cache.sets != 1 {
			t.Errorf("got %+v and %d cache writes", got, cache.sets)
		}
	})

	t.Run("returns the cached lot without reading the repository", func(t *testing.T) {
		v := newView(t, lotview.StatusActive)
		views := &fakeViews{}
		uc := getpubliclot.New(views, &fakeCache{item: v})

		got, err := uc.Execute(ctx, v.LotID())

		if err != nil || got != v || views.calls != 0 {
			t.Errorf("got %+v, %v and %d repository calls", got, err, views.calls)
		}
	})

	t.Run("returns a voided lot so the page can show its status", func(t *testing.T) {
		v := newView(t, lotview.StatusVoided)
		uc := getpubliclot.New(&fakeViews{view: v}, &fakeCache{})

		got, err := uc.Execute(ctx, v.LotID())

		if err != nil || got.Status() != lotview.StatusVoided {
			t.Errorf("got %+v and %v, want the voided lot", got, err)
		}
	})

	t.Run("rejects a default ID", func(t *testing.T) {
		uc := getpubliclot.New(&fakeViews{}, &fakeCache{})

		got, err := uc.Execute(ctx, uuid.Nil)

		assertCode(t, err, exception.CodeLotViewInvalidID)
		if !got.IsDefault() {
			t.Errorf("got %+v, want default", got)
		}
	})

	t.Run("reports a missing lot as not found", func(t *testing.T) {
		uc := getpubliclot.New(&fakeViews{}, &fakeCache{})

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeLotViewNotFound)
		if !got.IsDefault() {
			t.Errorf("got %+v, want default", got)
		}
	})

	t.Run("wraps a repository failure", func(t *testing.T) {
		uc := getpubliclot.New(&fakeViews{err: errBackend}, &fakeCache{})

		_, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})
}
