package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
)

var _ ports.StageViewRepository = (*StageViewRepository)(nil)

type stageKey struct {
	lotID uuid.UUID
	stage lotview.Stage
}

type StageViewRepository struct {
	mu    sync.RWMutex
	views map[stageKey]*stageview.StageView
}

func NewStageViewRepository() *StageViewRepository {
	return &StageViewRepository{views: make(map[stageKey]*stageview.StageView)}
}

func (r *StageViewRepository) Save(_ context.Context, view *stageview.StageView) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.views[stageKey{lotID: view.LotID(), stage: view.Stage()}] = view
	return nil
}

func (r *StageViewRepository) ListByLotID(_ context.Context, lotID uuid.UUID) ([]*stageview.StageView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	found := make([]*stageview.StageView, 0)
	for key, view := range r.views {
		if key.lotID == lotID {
			found = append(found, view)
		}
	}

	sort.Slice(found, func(i, j int) bool {
		return stageOrder(found[i].Stage()) < stageOrder(found[j].Stage())
	})
	return found, nil
}

func stageOrder(stage lotview.Stage) int {
	switch stage {
	case lotview.StageFarm:
		return 1
	case lotview.StageRoasting:
		return 2
	case lotview.StagePackaging:
		return 3
	default:
		return 0
	}
}
