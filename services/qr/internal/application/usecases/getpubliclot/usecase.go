package getpubliclot

import (
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/lotview"
)

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, *lotview.LotView]
}
