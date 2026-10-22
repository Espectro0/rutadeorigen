package getqrimage_test

import (
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/getqrimage"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestAssemblerToQuery(t *testing.T) {
	lotID := uuid.New()

	t.Run("parses every field", func(t *testing.T) {
		q, err := getqrimage.Assembler{}.ToQuery(request.GetQRImage{LotID: " " + lotID.String() + " ", Size: " 1024 ", Format: " SVG "})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if q.LotID != lotID || q.Size != qrcode.SizeLarge || q.Format != qrcode.FormatSVG {
			t.Errorf("got %+v", q)
		}
	})

	t.Run("uses the defaults when size and format are missing", func(t *testing.T) {
		q, err := getqrimage.Assembler{}.ToQuery(request.GetQRImage{LotID: lotID.String()})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if q.Size != qrcode.DefaultSize || q.Format != qrcode.DefaultFormat {
			t.Errorf("got size %d and format %q", q.Size, q.Format)
		}
	})

	invalid := []struct {
		name     string
		req      request.GetQRImage
		wantCode exception.Code
	}{
		{"empty lot id", request.GetQRImage{}, exception.CodeLotViewNotFound},
		{"malformed lot id", request.GetQRImage{LotID: "not-a-uuid"}, exception.CodeLotViewNotFound},
		{"nil lot id", request.GetQRImage{LotID: uuid.Nil.String()}, exception.CodeLotViewNotFound},
		{"non numeric size", request.GetQRImage{LotID: lotID.String(), Size: "big"}, exception.CodeQRCodeInvalidSize},
		{"unsupported size", request.GetQRImage{LotID: lotID.String(), Size: "300"}, exception.CodeQRCodeInvalidSize},
		{"negative size", request.GetQRImage{LotID: lotID.String(), Size: "-512"}, exception.CodeQRCodeInvalidSize},
		{"unknown format", request.GetQRImage{LotID: lotID.String(), Format: "jpg"}, exception.CodeQRCodeInvalidFormat},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			q, err := getqrimage.Assembler{}.ToQuery(tc.req)

			if !exception.HasCode(err, tc.wantCode) {
				t.Fatalf("got error %v, want code %q", err, tc.wantCode)
			}
			if q.LotID != uuid.Nil {
				t.Errorf("on error the query must be empty, got %+v", q)
			}
		})
	}
}

func TestAssemblerToDTO(t *testing.T) {
	t.Run("maps the image", func(t *testing.T) {
		image, err := qrcode.NewImage([]byte{1, 2, 3}, qrcode.FormatPNG, "LT-001")
		if err != nil {
			t.Fatalf("cannot build image: %v", err)
		}

		got := getqrimage.Assembler{}.ToDTO(image)

		if string(got.Data) != string([]byte{1, 2, 3}) || got.ContentType != "image/png" || got.Filename != "qr-LT-001.png" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("default and nil images return an empty response", func(t *testing.T) {
		if got := (getqrimage.Assembler{}).ToDTO(qrcode.DefaultImage()); got.Data != nil || got.ContentType != "" || got.Filename != "" {
			t.Errorf("default: got %+v, want empty response", got)
		}
		if got := (getqrimage.Assembler{}).ToDTO(nil); got.Data != nil || got.ContentType != "" || got.Filename != "" {
			t.Errorf("nil: got %+v, want empty response", got)
		}
	})
}
