package registerstage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/registerstage"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

type ctxKey struct{}

type fakeUseCase struct {
	result registerstageuc.Result
	err    error

	calls    int
	received registerstageuc.Command
	ctx      context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, cmd registerstageuc.Command) (registerstageuc.Result, error) {
	f.calls++
	f.received = cmd
	f.ctx = ctx
	if f.err != nil {
		return registerstageuc.Result{Event: lot.DefaultEvent()}, f.err
	}
	return f.result, nil
}

func registeredEvent(t *testing.T) *lot.Event {
	t.Helper()
	e, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       uuid.New(),
		HarvestDate: time.Now().AddDate(0, -1, 0),
		Stage:       lot.StageFarm,
		EventDate:   time.Now().AddDate(0, 0, -1),
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details:     lot.FarmDetails{DryingMethod: lot.DryingMethodSunPatio, DryingDays: 15, MoisturePercent: 11.5, ParchmentWeightKg: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestAdapterExecute(t *testing.T) {
	t.Run("sends the command to the use case and returns the mapped result", func(t *testing.T) {
		event := registeredEvent(t)
		uc := &fakeUseCase{result: registerstageuc.Result{Event: event, LotVersion: 4}}
		ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
		req := validRequest()

		got, err := registerstage.New(uc).Execute(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.received.LotID.String() != req.LotID || uc.received.ExpectedVersion != 3 || uc.received.Stage != lot.StageFarm {
			t.Errorf("use case received an unexpected command: %+v", uc.received)
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
		if got.LotVersion != 4 {
			t.Errorf("LotVersion: got %d, want 4", got.LotVersion)
		}
		if got.Event.ID != event.ID().String() || got.Event.Hash != event.Hash() {
			t.Errorf("unexpected event: %+v", got.Event)
		}
	})

	t.Run("an invalid request does not reach the use case", func(t *testing.T) {
		uc := &fakeUseCase{}
		req := validRequest()
		req.IfMatch = ""

		got, err := registerstage.New(uc).Execute(context.Background(), req)

		if !exception.HasCode(err, exception.CodeLotInvalidIfMatch) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotInvalidIfMatch)
		}
		if uc.calls != 0 {
			t.Errorf("use case calls: got %d, want 0", uc.calls)
		}
		assertEmpty(t, got)
	})

	t.Run("invalid details do not reach the use case", func(t *testing.T) {
		uc := &fakeUseCase{}
		req := validRequest()
		req.Details = nil

		got, err := registerstage.New(uc).Execute(context.Background(), req)

		if !exception.HasCode(err, exception.CodeStageInvalidDetails) {
			t.Errorf("got %v, want code %q", err, exception.CodeStageInvalidDetails)
		}
		if uc.calls != 0 {
			t.Errorf("use case calls: got %d, want 0", uc.calls)
		}
		assertEmpty(t, got)
	})

	t.Run("a use case error is returned with an empty response", func(t *testing.T) {
		want := exception.New(exception.CodeLotVersionConflict)
		uc := &fakeUseCase{err: want}

		got, err := registerstage.New(uc).Execute(context.Background(), validRequest())

		if !errors.Is(err, want) {
			t.Errorf("got %v, want %v", err, want)
		}
		if uc.calls != 1 {
			t.Errorf("use case calls: got %d, want 1", uc.calls)
		}
		assertEmpty(t, got)
	})
}

func assertEmpty(t *testing.T, got response.RegisteredStage) {
	t.Helper()
	if got.Event.ID != "" || got.LotVersion != 0 {
		t.Errorf("expected an empty response, got %+v", got)
	}
}
