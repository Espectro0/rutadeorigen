package renderer

import (
	"context"
	"runtime/debug"

	goqrcode "github.com/skip2/go-qrcode"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.QRRenderer = (*GoQRCodeRenderer)(nil)

const (
	modulePath      = "github.com/skip2/go-qrcode"
	versionFallback = "skip2/go-qrcode"
)

type GoQRCodeRenderer struct {
	version string
}

func NewGoQRCodeRenderer() *GoQRCodeRenderer {
	return &GoQRCodeRenderer{version: libraryVersion()}
}

func (r *GoQRCodeRenderer) LibraryVersion() string {
	return r.version
}

func (r *GoQRCodeRenderer) Render(_ context.Context, request ports.RenderRequest) ([]byte, error) {
	level, ok := recoveryLevel(request.ErrorCorrection)
	if !ok {
		return nil, exception.New(exception.CodeQRCodeInvalidErrorCorrection)
	}
	if !request.Size.IsValid() {
		return nil, exception.New(exception.CodeQRCodeInvalidSize)
	}

	code, err := goqrcode.New(request.Content, level)
	if err != nil {
		return nil, exception.New(exception.CodeQRCodeInvalidContent).WithCause(err)
	}

	switch request.Format {
	case qrcode.FormatPNG:
		data, err := code.PNG(request.Size.Pixels())
		if err != nil {
			return nil, exception.New(exception.CodeInternal).WithTechnical("encode png").WithCause(err)
		}
		return data, nil
	case qrcode.FormatSVG:
		return buildSVG(code.Bitmap(), request.Size.Pixels()), nil
	default:
		return nil, exception.New(exception.CodeQRCodeInvalidFormat)
	}
}

func recoveryLevel(ec qrcode.ErrorCorrection) (goqrcode.RecoveryLevel, bool) {
	switch ec {
	case qrcode.ErrorCorrectionLow:
		return goqrcode.Low, true
	case qrcode.ErrorCorrectionMedium:
		return goqrcode.Medium, true
	case qrcode.ErrorCorrectionQuartile:
		return goqrcode.High, true
	case qrcode.ErrorCorrectionHigh:
		return goqrcode.Highest, true
	default:
		return goqrcode.Low, false
	}
}

func libraryVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return versionFallback
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath && dep.Version != "" {
			return versionFallback + "@" + dep.Version
		}
	}
	return versionFallback
}
