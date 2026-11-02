package memory

import (
	"context"

	"rutadeorigen/processor/internal/application/output/ports"
)

var _ ports.TxManager = TxManager{}

type TxManager struct{}

func NewTxManager() TxManager { return TxManager{} }

func (TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
