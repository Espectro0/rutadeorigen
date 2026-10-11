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
)

func seedActors() []*actor.Actor {
	return []*actor.Actor{
		actor.Restore(seedProducerID, "seed-producer", actor.TypeProducer,
			"Productor de prueba", actor.StatusActive, time.Now().UTC()),
	}
}

func seedFarms() []*farm.Farm {
	return []*farm.Farm{
		farm.Restore(seedFarmID, seedProducerID, "Finca de prueba", 1800, time.Now().UTC()),
	}
}
