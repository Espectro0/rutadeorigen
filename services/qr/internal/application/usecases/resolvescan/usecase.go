package resolvescan

import (
	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/redirect"
)

type UseCase interface {
	usecases.UseCaseWithReturn[string, redirect.Resolution]
}
