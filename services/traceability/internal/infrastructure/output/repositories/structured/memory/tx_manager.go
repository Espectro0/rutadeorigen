package memory

import "context"

type TxManager struct{}

func NewTxManager() TxManager { return TxManager{} }

func (TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
