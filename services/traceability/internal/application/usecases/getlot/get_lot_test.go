package getlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/getlot"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errDatabase = errors.New("database is down")

type fakeActors struct {
	actor *actor.Actor
	err   error
}

func (f fakeActors) FindByID(_ context.Context, _ uuid.UUID) (*actor.Actor, error) {
	return f.actor, f.err
}

type fakeLots struct {
	lot *lot.Lot
	err error
}

func (f fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (f fakeLots) Save(_ context.Context, _ *lot.Lot) error { return nil }

func (f fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	if f.err != nil {
		return lot.Default(), f.err
	}
	return f.lot, nil
}

func (f fakeLots) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
	calls    int
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.calls++
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

func activeActor(t *testing.T) *actor.Actor {
	t.Helper()

	a, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS")
	if err != nil {
		t.Fatalf("cannot build actor: %v", err)
	}
	return a
}

func existingLot() *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageNone,
		Status:       lot.StatusActive,
		Version:      1,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})
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

func TestGetLotExecute(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())

	t.Run("returns the lot and authorizes the read", func(t *testing.T) {
		a := activeActor(t)
		l := existingLot()
		auth := &fakeAuthorizer{allowed: true}
		uc := getlot.New(fakeActors{actor: a}, fakeLots{lot: l}, auth)

		got, err := uc.Execute(ctx, l.ID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != l {
			t.Errorf("got %+v, want %+v", got, l)
		}
		if auth.action != ports.ActionReadLot {
			t.Errorf("action: got %q, want %q", auth.action, ports.ActionReadLot)
		}
		if auth.subject.ActorID != a.ID() || auth.subject.Role != a.Type() {
			t.Errorf("subject: got %+v", auth.subject)
		}
		wantResource := ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()}
		if auth.resource != wantResource {
			t.Errorf("resource: got %+v, want %+v", auth.resource, wantResource)
		}
	})

	t.Run("rejects an unknown actor before reading the lot", func(t *testing.T) {
		auth := &fakeAuthorizer{allowed: true}
		uc := getlot.New(fakeActors{actor: actor.Default()}, fakeLots{lot: existingLot()}, auth)

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeActorNotFound)
		if !got.IsDefault() || auth.calls != 0 {
			t.Errorf("got %+v and %d authorizer calls, want default lot and 0", got, auth.calls)
		}
	})

	t.Run("reports a missing lot as not found", func(t *testing.T) {
		auth := &fakeAuthorizer{allowed: true}
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: lot.Default()}, auth)

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeLotNotFound)
		if ex, _ := exception.From(err); ex.Kind() != exception.KindNotFound {
			t.Errorf("Kind: got %v, want %v", ex.Kind(), exception.KindNotFound)
		}
		if !got.IsDefault() || auth.calls != 0 {
			t.Errorf("got %+v and %d authorizer calls, want default lot and 0", got, auth.calls)
		}
	})

	t.Run("wraps a repository failure", func(t *testing.T) {
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{err: errDatabase}, &fakeAuthorizer{allowed: true})

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errDatabase) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
		if !got.IsDefault() {
			t.Errorf("got %+v, want default lot", got)
		}
	})

	t.Run("forbids a denied read", func(t *testing.T) {
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: existingLot()}, &fakeAuthorizer{allowed: false})

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeForbidden)
		if !got.IsDefault() {
			t.Errorf("got %+v, want default lot", got)
		}
	})

	t.Run("wraps an authorizer failure", func(t *testing.T) {
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: existingLot()}, &fakeAuthorizer{err: errDatabase})

		_, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errDatabase) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})
}
