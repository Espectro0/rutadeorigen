package ports

import (
	"context"

	"rutadeorigen/qr/internal/domain/redirect"
)

type RedirectRepository interface {
	ListActive(ctx context.Context) ([]*redirect.Rule, error)
}
