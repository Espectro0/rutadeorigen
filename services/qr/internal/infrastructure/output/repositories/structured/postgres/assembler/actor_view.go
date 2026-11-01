package assembler

import (
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.ActorView, *actorview.ActorView] = ActorView{}

type ActorView struct{}

func (ActorView) ToDomain(e entity.ActorView) *actorview.ActorView {
	return actorview.Restore(actorview.RestoreParams{
		ActorID:   e.ActorID,
		UserID:    e.UserID,
		Role:      actorview.ParseRole(e.Role),
		CreatedAt: e.CreatedAt.UTC(),
	})
}

func (ActorView) ToEntity(v *actorview.ActorView) entity.ActorView {
	if v.IsDefault() {
		return entity.ActorView{}
	}

	return entity.ActorView{
		ActorID:   v.ActorID(),
		UserID:    v.UserID(),
		Role:      v.Role().String(),
		CreatedAt: v.CreatedAt().UTC(),
	}
}
