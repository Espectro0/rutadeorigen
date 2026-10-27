package listlots_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/listlots"
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
	rows []*lot.Lot
	err  error

	calls    int
	filter   lot.Filter
	after    lot.Cursor
	received int
}

func (f *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (f *fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) Save(_ context.Context, _ *lot.Lot) error { return nil }

func (f *fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	return lot.Default(), nil
}

func (f *fakeLots) List(_ context.Context, filter lot.Filter, after lot.Cursor, limit int) ([]*lot.Lot, error) {
	f.calls++
	f.filter, f.after, f.received = filter, after, limit
	if f.err != nil {
		return nil, f.err
	}
	if len(f.rows) > limit {
		return f.rows[:limit], nil
	}
	return f.rows, nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, _ ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.action, a.resource = act, r
	return a.allowed, a.err
}

func activeActor(t *testing.T) *actor.Actor {
	t.Helper()

	a, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatalf("cannot build actor: %v", err)
	}
	return a
}

func lotCreatedAt(minutes int) *lot.Lot {
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
		CreatedAt:    time.Date(2026, time.September, 20, 14, minutes, 0, 0, time.UTC),
	})
}

func newRows(count int) []*lot.Lot {
	rows := make([]*lot.Lot, 0, count)
	for i := 0; i < count; i++ {
		rows = append(rows, lotCreatedAt(count-i))
	}
	return rows
}

func newUseCase(t *testing.T, lots *fakeLots, auth *fakeAuthorizer) *listlots.ListLots {
	t.Helper()

	return listlots.New(fakeActors{actor: activeActor(t)}, lots, auth)
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

func TestListLotsExecute(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())

	t.Run("returns the next cursor when there are more lots", func(t *testing.T) {
		rows := newRows(3)
		uc := newUseCase(t, &fakeLots{rows: rows}, &fakeAuthorizer{allowed: true})

		page, err := uc.Execute(ctx, listlots.Query{Limit: 2})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Lots) != 2 || page.Lots[0] != rows[0] || page.Lots[1] != rows[1] {
			t.Fatalf("got %d lots, want the first two", len(page.Lots))
		}
		if page.Next != lot.CursorOf(rows[1]) {
			t.Errorf("Next: got %+v, want the cursor of the last returned lot", page.Next)
		}
	})

	t.Run("has no next cursor when the lots fit in the page", func(t *testing.T) {
		uc := newUseCase(t, &fakeLots{rows: newRows(2)}, &fakeAuthorizer{allowed: true})

		page, err := uc.Execute(ctx, listlots.Query{Limit: 2})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Lots) != 2 || !page.Next.IsDefault() {
			t.Errorf("got %d lots and next %+v, want 2 and a default cursor", len(page.Lots), page.Next)
		}
	})

	t.Run("returns an empty page when nothing matches", func(t *testing.T) {
		uc := newUseCase(t, &fakeLots{}, &fakeAuthorizer{allowed: true})

		page, err := uc.Execute(ctx, listlots.Query{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Lots) != 0 || !page.Next.IsDefault() {
			t.Errorf("got %d lots and next %+v, want 0 and a default cursor", len(page.Lots), page.Next)
		}
	})

	t.Run("asks the repository for one extra lot", func(t *testing.T) {
		tests := []struct {
			name  string
			limit int
			want  int
		}{
			{"explicit limit", 5, 6},
			{"zero uses the default", 0, 21},
			{"negative uses the default", -3, 21},
			{"above the maximum is capped", 500, 101},
			{"the maximum is allowed", 100, 101},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				lots := &fakeLots{}
				uc := newUseCase(t, lots, &fakeAuthorizer{allowed: true})

				if _, err := uc.Execute(ctx, listlots.Query{Limit: tt.limit}); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if lots.received != tt.want {
					t.Errorf("repository limit: got %d, want %d", lots.received, tt.want)
				}
			})
		}
	})

	t.Run("passes the filter and the cursor to the repository", func(t *testing.T) {
		lots := &fakeLots{}
		uc := newUseCase(t, lots, &fakeAuthorizer{allowed: true})
		query := listlots.Query{
			Filter: lot.Filter{FarmID: uuid.New(), CustodianID: uuid.New()},
			After:  lot.CursorOf(lotCreatedAt(10)),
			Limit:  10,
		}

		if _, err := uc.Execute(ctx, query); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if lots.filter != query.Filter || lots.after != query.After {
			t.Errorf("repository got filter %+v and cursor %+v, want %+v and %+v",
				lots.filter, lots.after, query.Filter, query.After)
		}
	})

	t.Run("authorizes the read on the lot collection", func(t *testing.T) {
		auth := &fakeAuthorizer{allowed: true}
		uc := newUseCase(t, &fakeLots{}, auth)

		if _, err := uc.Execute(ctx, listlots.Query{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if auth.action != ports.ActionReadLot || auth.resource.Type != ports.ResourceLot {
			t.Errorf("got action %q on %+v", auth.action, auth.resource)
		}
	})

	t.Run("rejects an unknown actor without querying", func(t *testing.T) {
		lots := &fakeLots{}
		uc := listlots.New(fakeActors{actor: actor.Default()}, lots, &fakeAuthorizer{allowed: true})

		_, err := uc.Execute(ctx, listlots.Query{})

		assertCode(t, err, exception.CodeActorNotFound)
		if lots.calls != 0 {
			t.Errorf("repository calls: got %d, want 0", lots.calls)
		}
	})

	t.Run("forbids a denied read without querying", func(t *testing.T) {
		lots := &fakeLots{}
		uc := newUseCase(t, lots, &fakeAuthorizer{allowed: false})

		_, err := uc.Execute(ctx, listlots.Query{})

		assertCode(t, err, exception.CodeForbidden)
		if lots.calls != 0 {
			t.Errorf("repository calls: got %d, want 0", lots.calls)
		}
	})

	t.Run("wraps an authorizer failure", func(t *testing.T) {
		uc := newUseCase(t, &fakeLots{}, &fakeAuthorizer{err: errDatabase})

		_, err := uc.Execute(ctx, listlots.Query{})

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errDatabase) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})

	t.Run("wraps a repository failure", func(t *testing.T) {
		uc := newUseCase(t, &fakeLots{err: errDatabase}, &fakeAuthorizer{allowed: true})

		page, err := uc.Execute(ctx, listlots.Query{})

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errDatabase) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
		if len(page.Lots) != 0 {
			t.Errorf("got %d lots, want none", len(page.Lots))
		}
	})
}

func (f fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (f fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}
