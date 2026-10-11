package ports

import "context"

type InputPort[T any, R any] interface {
	Execute(ctx context.Context, input T) (R, error)
}
