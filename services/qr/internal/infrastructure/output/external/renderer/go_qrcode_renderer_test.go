package renderer_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"image/png"
	"io"
	"strings"
	"testing"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/infrastructure/output/external/renderer"
	"rutadeorigen/qr/internal/platform/exception"
)

func request(format qrcode.Format) ports.RenderRequest {
	return ports.RenderRequest{
		Content:         "https://rutadeorigen.co/l/0198a000-0000-7000-8000-000000000001",
		ErrorCorrection: qrcode.ErrorCorrectionQuartile,
		Size:            qrcode.SizeSmall,
		Format:          format,
	}
}

func TestRenderPNG(t *testing.T) {
	data, err := renderer.NewGoQRCodeRenderer().Render(context.Background(), request(qrcode.FormatPNG))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid png: %v", err)
	}
	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
		t.Errorf("unexpected bounds %v", img.Bounds())
	}
}

func TestRenderSVG(t *testing.T) {
	data, err := renderer.NewGoQRCodeRenderer().Render(context.Background(), request(qrcode.FormatSVG))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("not well-formed xml: %v", err)
		}
		if start, ok := token.(xml.StartElement); ok && root == "" {
			root = start.Name.Local
		}
	}
	if root != "svg" {
		t.Errorf("unexpected root element %q", root)
	}

	text := string(data)
	if !strings.Contains(text, `width="256"`) || !strings.Contains(text, `<path fill="#000000" d="M`) {
		t.Errorf("unexpected svg %.120s", text)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	r := renderer.NewGoQRCodeRenderer()

	first, _ := r.Render(context.Background(), request(qrcode.FormatSVG))
	second, _ := r.Render(context.Background(), request(qrcode.FormatSVG))

	if !bytes.Equal(first, second) {
		t.Error("expected identical output for identical input")
	}
}

func TestRenderAcceptsEveryErrorCorrection(t *testing.T) {
	for _, ec := range []qrcode.ErrorCorrection{
		qrcode.ErrorCorrectionLow, qrcode.ErrorCorrectionMedium, qrcode.ErrorCorrectionQuartile, qrcode.ErrorCorrectionHigh,
	} {
		req := request(qrcode.FormatPNG)
		req.ErrorCorrection = ec
		if _, err := renderer.NewGoQRCodeRenderer().Render(context.Background(), req); err != nil {
			t.Errorf("%s: unexpected error: %v", ec, err)
		}
	}
}

func TestRenderRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.RenderRequest)
		code   exception.Code
	}{
		{"error correction", func(r *ports.RenderRequest) { r.ErrorCorrection = qrcode.ErrorCorrectionUnknown }, exception.CodeQRCodeInvalidErrorCorrection},
		{"size", func(r *ports.RenderRequest) { r.Size = qrcode.SizeUnknown }, exception.CodeQRCodeInvalidSize},
		{"format", func(r *ports.RenderRequest) { r.Format = qrcode.FormatUnknown }, exception.CodeQRCodeInvalidFormat},
		{"empty content", func(r *ports.RenderRequest) { r.Content = "" }, exception.CodeQRCodeInvalidContent},
		{"oversized content", func(r *ports.RenderRequest) { r.Content = strings.Repeat("x", 5000) }, exception.CodeQRCodeInvalidContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := request(qrcode.FormatPNG)
			tt.mutate(&req)

			data, err := renderer.NewGoQRCodeRenderer().Render(context.Background(), req)

			if !exception.HasCode(err, tt.code) || data != nil {
				t.Errorf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestLibraryVersionIsNotEmpty(t *testing.T) {
	if renderer.NewGoQRCodeRenderer().LibraryVersion() == "" {
		t.Error("expected a library version")
	}
}
