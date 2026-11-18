package listpublicstages

import (
	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
)

type UseCase interface {
	usecases.UseCaseWithReturn[*lotview.LotView, []*stageview.StageView]
}
