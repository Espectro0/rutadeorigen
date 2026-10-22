package resolvescan_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/resolvescan"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/domain/redirect"
)

type fakeUseCase struct {
	result redirect.Resolution
	err    error

	received string
}

func (f *fakeUseCase) Execute(_ context.Context, path string) (redirect.Resolution, error) {
	f.received = path
	if f.err != nil {
		return redirect.Resolution{}, f.err
	}
	return f.result, nil
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the trimmed path and maps the resolution", func(t *testing.T) {
		lotID := uuid.New()
		uc := &fakeUseCase{result: redirect.Resolution{
			LotID:      lotID,
			Location:   "/lotes/" + lotID.String(),
			StatusCode: 301,
		}}

		got, err := resolvescan.New(uc).Execute(ctx, request.ResolveScan{Path: "  /l/" + lotID.String() + " "})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.received != "/l/"+lotID.String() {
			t.Errorf("use case received %q", uc.received)
		}
		if got.Location != "/lotes/"+lotID.String() || got.StatusCode != 301 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("returns the use case error and an empty response", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		got, err := resolvescan.New(uc).Execute(ctx, request.ResolveScan{Path: "/l/x"})

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if got.Location != "" || got.StatusCode != 0 {
			t.Errorf("got %+v, want an empty response", got)
		}
	})
}
