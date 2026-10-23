package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/qrcode"
)

var _ ports.QRCodeRepository = (*QRCodeRepository)(nil)

type QRCodeRepository struct {
	mu    sync.RWMutex
	codes map[uuid.UUID]*qrcode.QRCode
}

func NewQRCodeRepository() *QRCodeRepository {
	return &QRCodeRepository{codes: make(map[uuid.UUID]*qrcode.QRCode)}
}

func (r *QRCodeRepository) FindByLotID(_ context.Context, lotID uuid.UUID) (*qrcode.QRCode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	code, ok := r.codes[lotID]
	if !ok {
		return qrcode.Default(), nil
	}
	return code, nil
}

func (r *QRCodeRepository) Save(_ context.Context, code *qrcode.QRCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.codes[code.LotID()]; ok {
		return nil
	}
	r.codes[code.LotID()] = code
	return nil
}
