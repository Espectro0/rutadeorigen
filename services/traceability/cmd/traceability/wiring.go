package main

import (
	"log/slog"
	"net/http"

	createlotadapter "rutadeorigen/traceability/internal/application/input/adapters/createlot"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
)

func registerRoutes(mux *http.ServeMux, log *slog.Logger) {
	actors := memory.NewActorRepository(seedActors()...)
	farms := memory.NewFarmRepository(seedFarms()...)
	lots := memory.NewLotRepository()
	auth := authorizer.NewAllowAll()
	publisher := messaging.NewLogPublisher(log)

	createLotUseCase := createlotuc.New(actors, farms, lots, auth, publisher)
	createLotAdapter := createlotadapter.New(createLotUseCase)

	requireActor := func(next http.Handler) http.Handler {
		return controllers.RequireActor(log, next)
	}

	controllers.NewLotController(createLotAdapter, log).Register(mux, requireActor)
}
