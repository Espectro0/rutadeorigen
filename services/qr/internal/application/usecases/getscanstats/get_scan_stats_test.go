package getscanstats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/getscanstats"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeViews struct{ view *lotview.LotView }

func (v fakeViews) FindByID(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	if v.view == nil {
		return lotview.Default(), nil
	}
	return v.view, nil
}

func (v fakeViews) Save(context.Context, *lotview.LotView) error { return nil }

type fakeCache struct{}

func (fakeCache) Get(context.Context, uuid.UUID) (*lotview.LotView, error) {
	return lotview.Default(), nil
}
func (fakeCache) Set(context.Context, *lotview.LotView) error { return nil }
func (fakeCache) Delete(context.Context, uuid.UUID) error     { return nil }

type fakeScans struct {
	err   error
	calls int
	lotID uuid.UUID
	since time.Time
	stats *scan.Stats
}

func (s *fakeScans) Save(context.Context, *scan.Scan) error { return nil }

func (s *fakeScans) StatsByLot(_ context.Context, lotID uuid.UUID, since time.Time) (*scan.Stats, error) {
	s.calls++
	s.lotID, s.since = lotID, since
	if s.err != nil {
		return nil, s.err
	}
	return s.stats, nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	calls    int
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.calls++
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

type fixture struct {
	view    *lotview.LotView
	scans   *fakeScans
	auth    *fakeAuthorizer
	actorID uuid.UUID
	ctx     context.Context
	uc      *getscanstats.GetScanStats
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

func newFixture(t *testing.T) *fixture {
	t.Helper()

	view := newView(t)
	f := &fixture{
		view:    view,
		scans:   &fakeScans{stats: scan.NewStats(view.LotID(), 3, time.Time{}, time.Time{}, nil)},
		auth:    &fakeAuthorizer{allowed: true},
		actorID: uuid.New(),
	}
	f.ctx = requestcontext.WithActorRole(requestcontext.WithActorID(context.Background(), f.actorID), "organization")
	f.uc = getscanstats.New(fakeViews{view: view}, fakeCache{}, f.scans, f.auth)
	return f
}

func (f *fixture) query(days int) getscanstats.Query {
	return getscanstats.Query{LotID: f.view.LotID(), Days: days}
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

func TestGetScanStatsReturnsTheStatsOfTheLot(t *testing.T) {
	f := newFixture(t)
	before := time.Now().UTC().Truncate(24 * time.Hour)

	got, err := f.uc.Execute(f.ctx, f.query(7))

	if err != nil || got != f.scans.stats {
		t.Fatalf("expected the repository stats, got %+v and %v", got, err)
	}
	after := time.Now().UTC().Truncate(24 * time.Hour)
	wantSince := before.AddDate(0, 0, -6)
	if f.scans.lotID != f.view.LotID() || (!f.scans.since.Equal(wantSince) && !f.scans.since.Equal(after.AddDate(0, 0, -6))) {
		t.Errorf("unexpected query: lot %s since %v, want since %v", f.scans.lotID, f.scans.since, wantSince)
	}
}

func TestGetScanStatsAuthorizesThroughThePolicy(t *testing.T) {
	f := newFixture(t)

	if _, err := f.uc.Execute(f.ctx, f.query(30)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.auth.action != ports.ActionReadQRStats || f.auth.subject.ActorID != f.actorID || f.auth.subject.Role != actorview.RoleOrganization {
		t.Errorf("unexpected action %q or subject %+v", f.auth.action, f.auth.subject)
	}
	wantResource := ports.Resource{Type: ports.ResourceLot, ID: f.view.LotID(), CustodianID: f.view.CustodianID(), Stage: lotview.StagePackaging}
	if f.auth.resource != wantResource {
		t.Errorf("resource: got %+v, want %+v", f.auth.resource, wantResource)
	}
}

func TestGetScanStatsRejectsWhatTheUserMayNotSee(t *testing.T) {
	f := newFixture(t)
	f.auth.allowed = false

	got, err := f.uc.Execute(f.ctx, f.query(30))

	assertCode(t, err, exception.CodeForbidden)
	if got != nil || f.scans.calls != 0 {
		t.Error("nothing must be queried when the policy denies")
	}
}

func TestGetScanStatsRejectsRequestsWithoutActor(t *testing.T) {
	f := newFixture(t)

	_, err := f.uc.Execute(context.Background(), f.query(30))

	assertCode(t, err, exception.CodeUnauthorized)
	if f.auth.calls != 0 {
		t.Error("the policy must not run without an actor")
	}
}

func TestGetScanStatsRejectsInvalidDays(t *testing.T) {
	for _, days := range []int{-1, 0, scan.MaxStatsDays + 1} {
		f := newFixture(t)

		_, err := f.uc.Execute(f.ctx, f.query(days))

		assertCode(t, err, exception.CodeStatsInvalidDays)
		if f.scans.calls != 0 {
			t.Errorf("days %d: nothing must be queried", days)
		}
	}
}

func TestGetScanStatsFailsForAnUnknownLot(t *testing.T) {
	f := newFixture(t)
	uc := getscanstats.New(fakeViews{}, fakeCache{}, f.scans, f.auth)

	_, err := uc.Execute(f.ctx, f.query(30))

	assertCode(t, err, exception.CodeLotViewNotFound)
	if f.auth.calls != 0 {
		t.Error("the policy must not run for an unknown lot")
	}
}

func TestGetScanStatsWrapsPolicyAndRepositoryFailures(t *testing.T) {
	f := newFixture(t)
	f.auth.err = errBackend
	if _, err := f.uc.Execute(f.ctx, f.query(30)); !errors.Is(err, errBackend) {
		t.Errorf("policy: expected the backend error, got %v", err)
	}

	f = newFixture(t)
	f.scans.err = errBackend
	if _, err := f.uc.Execute(f.ctx, f.query(30)); !errors.Is(err, errBackend) {
		t.Errorf("repository: expected the backend error, got %v", err)
	}
}
