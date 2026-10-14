package assembler

import (
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.Farm, *farm.Farm] = Farm{}

type Farm struct{}

func (Farm) ToDomain(e entity.Farm) *farm.Farm {
	return farm.Restore(e.ID, e.ProducerID, e.Name, e.Altitude, e.CreatedAt.UTC())
}

func (Farm) ToEntity(f *farm.Farm) entity.Farm {
	if f.IsDefault() {
		return entity.Farm{}
	}

	return entity.Farm{
		ID:         f.ID(),
		ProducerID: f.ProducerID(),
		Name:       f.Name(),
		Altitude:   f.Altitude(),
		CreatedAt:  f.CreatedAt().UTC(),
	}
}
