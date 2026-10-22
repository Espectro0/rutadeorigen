package resolvescan_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/resolvescan"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeRules struct {
	rules []*redirect.Rule
	err   error
}

func (r fakeRules) ListActive(_ context.Context) ([]*redirect.Rule, error) {
	return r.rules, r.err
}

type fakeViews struct {
	view *lotview.LotView
	err  error
}

func (v fakeViews) FindByID(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	if v.err != nil {
		return lotview.Default(), v.err
	}
	if v.view == nil {
		return lotview.Default(), nil
	}
	return v.view, nil
}

func (v fakeViews) Save(_ context.Context, _ *lotview.LotView) error { return nil }

type fakeCache struct{}

func (fakeCache) Get(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	return lotview.Default(), nil
}

func (fakeCache) Set(_ context.Context, _ *lotview.LotView) error { return nil }

func (fakeCache) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type fakeScans struct {
	err   error
	saved []*scan.Scan
}

func (s *fakeScans) Save(_ context.Context, sc *scan.Scan) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, sc)
	return nil
}

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

func newRule(t *testing.T, prefix, template string, status int) *redirect.Rule {
	t.Helper()

	r, err := redirect.New(prefix, template, status)
	if err != nil {
		t.Fatalf("cannot build rule: %v", err)
	}
	return r
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

func TestResolveScanExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves the redirect and records the scan", func(t *testing.T) {
		v := newView(t)
		scans := &fakeScans{}
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{view: v}, fakeCache{}, scans)

		got, err := uc.Execute(ctx, "/l/"+v.LotID().String())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.LotID != v.LotID() || got.Location != "/lotes/"+v.LotID().String() || got.StatusCode != 301 {
			t.Errorf("unexpected resolution: %+v", got)
		}
		if len(scans.saved) != 1 || scans.saved[0].LotID() != v.LotID() {
			t.Errorf("expected one scan for the lot, got %+v", scans.saved)
		}
	})

	t.Run("uses the rule that matches the path", func(t *testing.T) {
		v := newView(t)
		rules := []*redirect.Rule{
			newRule(t, "/x/", "/other/{id}", 302),
			newRule(t, "/l/", "https://front.rutadeorigen.co/lotes/{id}", 308),
		}
		uc := resolvescan.New(fakeRules{rules: rules}, fakeViews{view: v}, fakeCache{}, &fakeScans{})

		got, err := uc.Execute(ctx, "/l/"+v.LotID().String())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Location != "https://front.rutadeorigen.co/lotes/"+v.LotID().String() || got.StatusCode != 308 {
			t.Errorf("unexpected resolution: %+v", got)
		}
	})

	t.Run("skips an inactive rule", func(t *testing.T) {
		v := newView(t)
		inactive := redirect.Restore(redirect.RestoreParams{
			ID: uuid.New(), SourcePrefix: "/l/", TargetTemplate: "/old/{id}", StatusCode: 301, Active: false,
		})
		rules := []*redirect.Rule{inactive, newRule(t, "/l/", "/lotes/{id}", 301)}
		uc := resolvescan.New(fakeRules{rules: rules}, fakeViews{view: v}, fakeCache{}, &fakeScans{})

		got, err := uc.Execute(ctx, "/l/"+v.LotID().String())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Location != "/lotes/"+v.LotID().String() {
			t.Errorf("Location: got %q", got.Location)
		}
	})

	t.Run("still redirects when the scan cannot be saved", func(t *testing.T) {
		v := newView(t)
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{view: v}, fakeCache{}, &fakeScans{err: errBackend})

		got, err := uc.Execute(ctx, "/l/"+v.LotID().String())

		if err != nil || got.LotID != v.LotID() {
			t.Errorf("got %+v and %v, want the resolution and no error", got, err)
		}
	})

	t.Run("reports a path without a matching rule as not found", func(t *testing.T) {
		scans := &fakeScans{}
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{}, fakeCache{}, scans)

		got, err := uc.Execute(ctx, "/z/"+uuid.NewString())

		assertCode(t, err, exception.CodeRedirectNotFound)
		if got.StatusCode != 0 || len(scans.saved) != 0 {
			t.Errorf("got %+v and %d scans, want empty and 0", got, len(scans.saved))
		}
	})

	t.Run("reports an empty rule list as not found", func(t *testing.T) {
		uc := resolvescan.New(fakeRules{}, fakeViews{}, fakeCache{}, &fakeScans{})

		_, err := uc.Execute(ctx, "/l/"+uuid.NewString())

		assertCode(t, err, exception.CodeRedirectNotFound)
	})

	t.Run("wraps a rules failure", func(t *testing.T) {
		uc := resolvescan.New(fakeRules{err: errBackend}, fakeViews{}, fakeCache{}, &fakeScans{})

		_, err := uc.Execute(ctx, "/l/"+uuid.NewString())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})

	t.Run("rejects a malformed lot ID without recording a scan", func(t *testing.T) {
		scans := &fakeScans{}
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{}, fakeCache{}, scans)

		_, err := uc.Execute(ctx, "/l/not-a-uuid")

		assertCode(t, err, exception.CodeLotViewInvalidID)
		if len(scans.saved) != 0 {
			t.Errorf("expected no scans, got %d", len(scans.saved))
		}
	})

	t.Run("reports an unknown lot as not found without recording a scan", func(t *testing.T) {
		scans := &fakeScans{}
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{}, fakeCache{}, scans)

		_, err := uc.Execute(ctx, "/l/"+uuid.NewString())

		assertCode(t, err, exception.CodeLotViewNotFound)
		if len(scans.saved) != 0 {
			t.Errorf("expected no scans, got %d", len(scans.saved))
		}
	})

	t.Run("wraps a lot lookup failure", func(t *testing.T) {
		uc := resolvescan.New(fakeRules{rules: []*redirect.Rule{newRule(t, "/l/", "/lotes/{id}", 301)}}, fakeViews{err: errBackend}, fakeCache{}, &fakeScans{})

		_, err := uc.Execute(ctx, "/l/"+uuid.NewString())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})
}
