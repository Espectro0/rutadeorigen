package getscanstats

import (
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/scan"
)

type Query struct {
	LotID uuid.UUID
	Days  int
}

type UseCase interface {
	usecases.UseCaseWithReturn[Query, *scan.Stats]
}
