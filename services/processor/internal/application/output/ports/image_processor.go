package ports

import (
	"context"

	"rutadeorigen/processor/internal/domain/picture"
)

type ProcessedImage struct {
	Final     []byte
	Thumbnail []byte
}

type ImageProcessor interface {
	Inspect(ctx context.Context, data []byte) (picture.Info, error)
	Process(ctx context.Context, data []byte, info picture.Info, thumbnailSide int) (ProcessedImage, error)
}
