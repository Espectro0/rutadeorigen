package main

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
)

var (
	seedProducerID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	seedFarmID     = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	seedRoasterID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	seedBrandID    = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	seedEmail      = "seed@rutadeorigen.com"
	seedRoasterEmail = "seed-roaster@rutadeorigen.com"
	seedBrandEmail   = "seed-brand@rutadeorigen.com"
	seedCreatedAt  = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
)

func seedActors() []*actor.Actor {
	return []*actor.Actor{
		actor.Restore(seedProducerID, "seed-producer", actor.TypeProducer,
			"Productor de prueba", seedEmail, actor.StatusActive, seedCreatedAt),
		actor.Restore(seedRoasterID, "seed-roaster", actor.TypeRoaster,
			"Tostador de prueba", seedRoasterEmail, actor.StatusActive, seedCreatedAt),
		actor.Restore(seedBrandID, "seed-brand", actor.TypeBrand,
			"Marca de prueba", seedBrandEmail, actor.StatusActive, seedCreatedAt),
	}
}

func seedFarms() []*farm.Farm {
	return []*farm.Farm{
		farm.Restore(seedFarmID, seedProducerID, "Finca de prueba", 1800, seedCreatedAt),
	}
}
