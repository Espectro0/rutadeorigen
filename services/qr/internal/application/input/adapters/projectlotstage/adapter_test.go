package projectlotstage_test

import (
	"context"
	"errors"
	"testing"

	"rutadeorigen/qr/internal/application/input/adapters/projectlotstage"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotstageuc "rutadeorigen/qr/internal/application/usecases/projectlotstage"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

type fakeUseCase struct {
	err error

	calls    int
	received projectlotstageuc.Command
}

func (f *fakeUseCase) Execute(_ context.Context, cmd projectlotstageuc.Command) error {
	f.calls++
	f.received = cmd
	return f.err
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the assembled command to the use case", func(t *testing.T) {
		req := validRequest()
		uc := &fakeUseCase{}

		res, err := projectlotstage.New(uc).Execute(ctx, req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != (response.Empty{}) {
			t.Errorf("got %+v, want an empty response", res)
		}
		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.received.MessageID.String() != req.MessageID || uc.received.LotID.String() != req.Data.LotID {
			t.Errorf("use case received %+v", uc.received)
		}
		if uc.received.Stage != lotview.StageRoasting || uc.received.Version != 3 || uc.received.EventType != "lot.stage_registered" {
			t.Errorf("use case received %+v", uc.received)
		}
	})

	t.Run("passes the context to the use case", func(t *testing.T) {
		type key struct{}
		var got context.Context
		uc := &contextUseCase{capture: func(ctx context.Context) { got = ctx }}
		ctx := context.WithValue(ctx, key{}, "marked")

		if _, err := projectlotstage.New(uc).Execute(ctx, validRequest()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got == nil || got.Value(key{}) != "marked" {
			t.Error("the context must reach the use case")
		}
	})

	t.Run("rejects an invalid message without calling the use case", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(r *request.ProjectLotStage)
		}{
			{"message ID", func(r *request.ProjectLotStage) { r.MessageID = "not-a-uuid" }},
			{"lot ID", func(r *request.ProjectLotStage) { r.Data.LotID = "not-a-uuid" }},
			{"occurred at", func(r *request.ProjectLotStage) { r.OccurredAt = "yesterday" }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := validRequest()
				tt.mutate(&req)
				uc := &fakeUseCase{}

				_, err := projectlotstage.New(uc).Execute(ctx, req)

				if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
					t.Errorf("got %v, want %s", err, exception.CodeEventInvalidPayload)
				}
				if uc.calls != 0 {
					t.Errorf("use case calls: got %d, want 0", uc.calls)
				}
			})
		}
	})

	t.Run("returns the use case error", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		res, err := projectlotstage.New(uc).Execute(ctx, validRequest())

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if res != (response.Empty{}) {
			t.Errorf("got %+v, want an empty response", res)
		}
	})

	t.Run("keeps the code of a classified use case error", func(t *testing.T) {
		uc := &fakeUseCase{err: exception.New(exception.CodeLotViewNotFound)}

		_, err := projectlotstage.New(uc).Execute(ctx, validRequest())

		if !exception.HasCode(err, exception.CodeLotViewNotFound) {
			t.Errorf("got %v, want %s", err, exception.CodeLotViewNotFound)
		}
	})
}

type contextUseCase struct {
	capture func(ctx context.Context)
}

func (c *contextUseCase) Execute(ctx context.Context, _ projectlotstageuc.Command) error {
	c.capture(ctx)
	return nil
}
