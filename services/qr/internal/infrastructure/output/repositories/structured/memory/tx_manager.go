package memory

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
)

var _ ports.TxManager = (*TxManager)(nil)

type TxManager struct{}

func NewTxManager() *TxManager {
	return &TxManager{}
}

func (*TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
