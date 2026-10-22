package validators_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/validators"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeCache struct {
	item   *lotview.LotView
	getErr error
	setErr error
	sets   []*lotview.LotView
}

func (c *fakeCache) Get(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	if c.getErr != nil {
		return lotview.Default(), c.getErr
	}
	if c.item == nil {
		return lotview.Default(), nil
	}
	return c.item, nil
}

func (c *fakeCache) Set(_ context.Context, v *lotview.LotView) error {
	c.sets = append(c.sets, v)
	return c.setErr
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

func newView(t *testing.T) *lotview.LotView {
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
		CurrentStage: lotview.StagePackaging,
		Status:       lotview.StatusActive,
		Version:      1,
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

func TestLotViewExists(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects a default ID without touching cache or repository", func(t *testing.T) {
		cache, views := &fakeCache{}, &fakeViews{}

		got, err := validators.LotViewExists(ctx, cache, views, uuid.Nil)

		assertCode(t, err, exception.CodeLotViewInvalidID)
		if !got.IsDefault() || views.calls != 0 {
			t.Errorf("got %+v and %d repository calls, want default and 0", got, views.calls)
		}
	})

	t.Run("returns the cached view without reading the repository", func(t *testing.T) {
		v := newView(t)
		cache, views := &fakeCache{item: v}, &fakeViews{}

		got, err := validators.LotViewExists(ctx, cache, views, v.LotID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != v || views.calls != 0 || len(cache.sets) != 0 {
			t.Errorf("got %+v, %d repository calls and %d cache writes", got, views.calls, len(cache.sets))
		}
	})

	t.Run("reads the repository on a cache miss and fills the cache", func(t *testing.T) {
		v := newView(t)
		cache, views := &fakeCache{}, &fakeViews{view: v}

		got, err := validators.LotViewExists(ctx, cache, views, v.LotID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != v || views.calls != 1 {
			t.Errorf("got %+v and %d repository calls", got, views.calls)
		}
		if len(cache.sets) != 1 || cache.sets[0] != v {
			t.Errorf("expected the view to be cached, got %v", cache.sets)
		}
	})

	t.Run("falls back to the repository when the cache fails", func(t *testing.T) {
		v := newView(t)
		cache, views := &fakeCache{getErr: errBackend}, &fakeViews{view: v}

		got, err := validators.LotViewExists(ctx, cache, views, v.LotID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != v || views.calls != 1 {
			t.Errorf("got %+v and %d repository calls", got, views.calls)
		}
	})

	t.Run("ignores a cache write failure", func(t *testing.T) {
		v := newView(t)
		cache, views := &fakeCache{setErr: errBackend}, &fakeViews{view: v}

		got, err := validators.LotViewExists(ctx, cache, views, v.LotID())

		if err != nil || got != v {
			t.Errorf("got %+v and %v, want the view and no error", got, err)
		}
	})

	t.Run("reports a missing view as not found", func(t *testing.T) {
		cache, views := &fakeCache{}, &fakeViews{}

		got, err := validators.LotViewExists(ctx, cache, views, uuid.New())

		assertCode(t, err, exception.CodeLotViewNotFound)
		if ex, _ := exception.From(err); ex.Kind() != exception.KindNotFound {
			t.Errorf("Kind: got %v, want %v", ex.Kind(), exception.KindNotFound)
		}
		if !got.IsDefault() || len(cache.sets) != 0 {
			t.Errorf("got %+v and %d cache writes, want default and 0", got, len(cache.sets))
		}
	})

	t.Run("wraps a repository failure", func(t *testing.T) {
		cache, views := &fakeCache{}, &fakeViews{err: errBackend}

		got, err := validators.LotViewExists(ctx, cache, views, uuid.New())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
		if !got.IsDefault() {
			t.Errorf("got %+v, want default", got)
		}
	})
}
