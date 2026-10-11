package usecases

import "context"

type UseCaseWithReturn[I, O any] interface {
	Execute(ctx context.Context, input I) (O, error)
}

type UseCaseWithoutReturn[I any] interface {
	Execute(ctx context.Context, input I) error
}
