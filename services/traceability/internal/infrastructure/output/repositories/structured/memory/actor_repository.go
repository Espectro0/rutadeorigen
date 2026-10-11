package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
)

var _ ports.ActorRepository = (*ActorRepository)(nil)

type ActorRepository struct {
	mu     sync.RWMutex
	actors map[uuid.UUID]*actor.Actor
}

func NewActorRepository(seed ...*actor.Actor) *ActorRepository {
	r := &ActorRepository{actors: make(map[uuid.UUID]*actor.Actor)}
	for _, a := range seed {
		if !a.IsDefault() {
			r.actors[a.ID()] = a
		}
	}
	return r
}

func (r *ActorRepository) FindByID(_ context.Context, id uuid.UUID) (*actor.Actor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.actors[id]
	if !ok {
		return actor.Default(), nil
	}
	return a, nil
}
