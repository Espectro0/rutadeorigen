package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
)

var _ ports.ActorViewRepository = (*ActorViewRepository)(nil)

type ActorViewRepository struct {
	mu     sync.RWMutex
	byUser map[string]*actorview.ActorView
	byID   map[uuid.UUID]*actorview.ActorView
}

func NewActorViewRepository() *ActorViewRepository {
	return &ActorViewRepository{
		byUser: make(map[string]*actorview.ActorView),
		byID:   make(map[uuid.UUID]*actorview.ActorView),
	}
}

func (r *ActorViewRepository) FindByID(_ context.Context, id uuid.UUID) (*actorview.ActorView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	view, ok := r.byID[id]
	if !ok {
		return actorview.Default(), nil
	}
	return view, nil
}

func (r *ActorViewRepository) FindByUserID(_ context.Context, userID string) (*actorview.ActorView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	view, ok := r.byUser[userID]
	if !ok {
		return actorview.Default(), nil
	}
	return view, nil
}

func (r *ActorViewRepository) Save(_ context.Context, view *actorview.ActorView) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if previous, ok := r.byUser[view.UserID()]; ok {
		delete(r.byID, previous.ActorID())
	}
	r.byUser[view.UserID()] = view
	r.byID[view.ActorID()] = view
	return nil
}
