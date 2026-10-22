package getqrimage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/getqrimage"
	"rutadeorigen/qr/internal/application/input/dto/request"
	getqrimageuc "rutadeorigen/qr/internal/application/usecases/getqrimage"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

type fakeUseCase struct {
	result *qrcode.Image
	err    error

	calls    int
	received getqrimageuc.Query
}

func (f *fakeUseCase) Execute(_ context.Context, query getqrimageuc.Query) (*qrcode.Image, error) {
	f.calls++
	f.received = query
	if f.err != nil {
		return qrcode.DefaultImage(), f.err
	}
	return f.result, nil
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the parsed query and maps the image", func(t *testing.T) {
		lotID := uuid.New()
		image, err := qrcode.NewImage([]byte{9, 9}, qrcode.FormatSVG, "LT-007")
		if err != nil {
			t.Fatalf("cannot build image: %v", err)
		}
		uc := &fakeUseCase{result: image}

		got, err := getqrimage.New(uc).Execute(ctx, request.GetQRImage{LotID: lotID.String(), Size: "256", Format: "svg"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := getqrimageuc.Query{LotID: lotID, Size: qrcode.SizeSmall, Format: qrcode.FormatSVG}
		if uc.received != want {
			t.Errorf("use case received %+v, want %+v", uc.received, want)
		}
		if got.ContentType != "image/svg+xml" || got.Filename != "qr-LT-007.svg" || len(got.Data) != 2 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("rejects an invalid request without calling the use case", func(t *testing.T) {
		uc := &fakeUseCase{}

		_, err := getqrimage.New(uc).Execute(ctx, request.GetQRImage{LotID: uuid.NewString(), Size: "300"})

		if !exception.HasCode(err, exception.CodeQRCodeInvalidSize) {
			t.Errorf("got %v, want %s", err, exception.CodeQRCodeInvalidSize)
		}
		if uc.calls != 0 {
			t.Errorf("use case calls: got %d, want 0", uc.calls)
		}
	})

	t.Run("returns the use case error and an empty response", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		got, err := getqrimage.New(uc).Execute(ctx, request.GetQRImage{LotID: uuid.NewString()})

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if got.Data != nil || got.ContentType != "" {
			t.Errorf("got %+v, want an empty response", got)
		}
	})
}
