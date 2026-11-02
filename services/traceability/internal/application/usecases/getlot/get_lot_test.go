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

func (f fakeLots) UpdateStatus(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f fakeLots) UpdateCustodian(_ context.Context, _ *lot.Lot, _ int) error {
	return nil
}

func (f fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error { return nil }

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
	only     map[ports.Action]bool
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
	calls    int
	asked    []ports.Action
	stages   map[ports.Action]lot.Stage
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	if a.calls == 0 {
		a.subject, a.action, a.resource = s, act, r
	}
	a.calls++
	a.asked = append(a.asked, act)
	if a.stages == nil {
		a.stages = map[ports.Action]lot.Stage{}
	}
	a.stages[act] = r.Stage
	if a.err != nil {
		return false, a.err
	}
	if a.only != nil {
		return act == ports.ActionReadLot || a.only[act], nil
	}
	return a.allowed, nil
}

func activeActor(t *testing.T) *actor.Actor {
	t.Helper()

	a, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
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
		if got.Lot != l {
			t.Errorf("got %+v, want %+v", got.Lot, l)
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
		if !got.Lot.IsDefault() || auth.calls != 0 {
			t.Errorf("got %+v and %d authorizer calls, want default lot and 0", got.Lot, auth.calls)
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
		if !got.Lot.IsDefault() || auth.calls != 0 {
			t.Errorf("got %+v and %d authorizer calls, want default lot and 0", got.Lot, auth.calls)
		}
	})

	t.Run("wraps a repository failure", func(t *testing.T) {
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{err: errDatabase}, &fakeAuthorizer{allowed: true})

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errDatabase) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
		if !got.Lot.IsDefault() {
			t.Errorf("got %+v, want default lot", got.Lot)
		}
	})

	t.Run("forbids a denied read", func(t *testing.T) {
		uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: existingLot()}, &fakeAuthorizer{allowed: false})

		got, err := uc.Execute(ctx, uuid.New())

		assertCode(t, err, exception.CodeForbidden)
		if !got.Lot.IsDefault() {
			t.Errorf("got %+v, want default lot", got.Lot)
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

func (f fakeActors) ListByType(_ context.Context, _ actor.Type) ([]*actor.Actor, error) {
	return nil, nil
}

func (f fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (f fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}

func lotAt(stage lot.Stage, status lot.Status, custodian uuid.UUID) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  custodian,
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: stage,
		Status:       status,
		Version:      3,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})
}

func TestGetLotAllowedActionsFollowTheLotState(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())

	tests := []struct {
		name   string
		lot    *lot.Lot
		asked  []ports.Action
		stages map[ports.Action]lot.Stage
	}{
		{
			name:   "a lot without events can register the farm stage or be voided",
			lot:    lotAt(lot.StageNone, lot.StatusActive, uuid.New()),
			asked:  []ports.Action{ports.ActionRegisterStage, ports.ActionVoidLot},
			stages: map[ports.Action]lot.Stage{ports.ActionRegisterStage: lot.StageFarm},
		},
		{
			name:   "a lot at the farm stage can register roasting, transfer, receive evidence or be voided",
			lot:    lotAt(lot.StageFarm, lot.StatusActive, uuid.New()),
			asked:  []ports.Action{ports.ActionRegisterStage, ports.ActionTransferLot, ports.ActionUploadEvidence, ports.ActionVoidLot},
			stages: map[ports.Action]lot.Stage{ports.ActionRegisterStage: lot.StageRoasting},
		},
		{
			name:   "a lot at roasting can register packaging, transfer, receive evidence or be voided",
			lot:    lotAt(lot.StageRoasting, lot.StatusActive, uuid.New()),
			asked:  []ports.Action{ports.ActionRegisterStage, ports.ActionTransferLot, ports.ActionUploadEvidence, ports.ActionVoidLot},
			stages: map[ports.Action]lot.Stage{ports.ActionRegisterStage: lot.StagePackaging},
		},
		{
			name:  "a packaged lot can receive evidence, be voided or have its QR generated",
			lot:   lotAt(lot.StagePackaging, lot.StatusActive, uuid.New()),
			asked: []ports.Action{ports.ActionUploadEvidence, ports.ActionVoidLot, ports.ActionGenerateQR},
		},
		{
			name:  "a voided lot offers no action",
			lot:   lotAt(lot.StageFarm, lot.StatusVoided, uuid.New()),
			asked: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuthorizer{allowed: true}
			uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: tt.lot}, auth)

			got, err := uc.Execute(ctx, tt.lot.ID())

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := append([]ports.Action{}, tt.asked...)
			if !equalActions(got.AllowedActions, want) {
				t.Errorf("allowed actions: got %v, want %v", got.AllowedActions, want)
			}
			if got.AllowedActions == nil {
				t.Error("allowed actions must never be nil")
			}
			for action, stage := range tt.stages {
				if auth.stages[action] != stage {
					t.Errorf("%s was evaluated against stage %q, want %q", action, auth.stages[action], stage)
				}
			}
		})
	}
}

func TestGetLotAllowedActionsKeepOnlyWhatThePolicyAllows(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())
	l := lotAt(lot.StageFarm, lot.StatusActive, uuid.New())
	auth := &fakeAuthorizer{only: map[ports.Action]bool{ports.ActionTransferLot: true}}
	uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: l}, auth)

	got, err := uc.Execute(ctx, l.ID())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalActions(got.AllowedActions, []ports.Action{ports.ActionTransferLot}) {
		t.Errorf("allowed actions: got %v, want only %s", got.AllowedActions, ports.ActionTransferLot)
	}
}

func TestGetLotAllowedActionsAreEmptyWhenTheRoleCanOnlyRead(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())
	l := lotAt(lot.StageFarm, lot.StatusActive, uuid.New())
	auth := &fakeAuthorizer{only: map[ports.Action]bool{}}
	uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: l}, auth)

	got, err := uc.Execute(ctx, l.ID())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lot != l || got.AllowedActions == nil || len(got.AllowedActions) != 0 {
		t.Errorf("got %+v, want the lot with an empty non nil list", got)
	}
}

func TestGetLotFailsWhenAnActionPolicyCannotBeEvaluated(t *testing.T) {
	ctx := requestcontext.WithActorID(context.Background(), uuid.New())
	l := lotAt(lot.StageFarm, lot.StatusActive, uuid.New())
	auth := &failAfterReadAuthorizer{}
	uc := getlot.New(fakeActors{actor: activeActor(t)}, fakeLots{lot: l}, auth)

	got, err := uc.Execute(ctx, l.ID())

	assertCode(t, err, exception.CodeInternal)
	if !errors.Is(err, errDatabase) {
		t.Errorf("the cause must be preserved, got %v", err)
	}
	if !got.Lot.IsDefault() || got.AllowedActions != nil {
		t.Errorf("got %+v, want the failed result", got)
	}
}

type failAfterReadAuthorizer struct{}

func (failAfterReadAuthorizer) IsAllowed(_ context.Context, _ ports.Subject, act ports.Action, _ ports.Resource) (bool, error) {
	if act == ports.ActionReadLot {
		return true, nil
	}
	return false, errDatabase
}

func equalActions(got, want []ports.Action) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
