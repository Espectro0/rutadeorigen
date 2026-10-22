package projectlotcreated_test

import (
	"context"
	"errors"
	"testing"

	"rutadeorigen/qr/internal/application/input/adapters/projectlotcreated"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
	"rutadeorigen/qr/internal/platform/exception"
)

type fakeUseCase struct {
	err error

	calls    int
	received projectlotcreateduc.Command
}

func (f *fakeUseCase) Execute(_ context.Context, cmd projectlotcreateduc.Command) error {
	f.calls++
	f.received = cmd
	return f.err
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the assembled command to the use case", func(t *testing.T) {
		req := validRequest()
		uc := &fakeUseCase{}

		_, err := projectlotcreated.New(uc).Execute(ctx, req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.received.MessageID.String() != req.MessageID || uc.received.Lot.LotID.String() != req.Data.LotID {
			t.Errorf("use case received %+v", uc.received)
		}
	})

	t.Run("rejects an invalid message without calling the use case", func(t *testing.T) {
		req := validRequest()
		req.MessageID = "not-a-uuid"
		uc := &fakeUseCase{}

		_, err := projectlotcreated.New(uc).Execute(ctx, req)

		if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
			t.Errorf("got %v, want %s", err, exception.CodeEventInvalidPayload)
		}
		if uc.calls != 0 {
			t.Errorf("use case calls: got %d, want 0", uc.calls)
		}
	})

	t.Run("returns the use case error", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		_, err := projectlotcreated.New(uc).Execute(ctx, validRequest())

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
	})
}
