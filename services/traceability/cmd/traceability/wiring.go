package main

import (
	"log/slog"
	"net/http"
	"rutadeorigen/lib/catalog"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"

	createlotadapter "rutadeorigen/traceability/internal/application/input/adapters/createlot"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerRoutes(mux *http.ServeMux, log *slog.Logger, pool *pgxpool.Pool, catalog *catalog.Catalog) {
	actors, farms, lots := repositories(pool)
	auth := authorizer.NewAllowAll()
	publisher := messaging.NewLogPublisher(log)
	errs := controllers.NewErrorResponder(log, catalog)

	createLotUseCase := createlotuc.New(actors, farms, lots, auth, publisher)
	createLotAdapter := createlotadapter.New(createLotUseCase)

	requireActor := func(next http.Handler) http.Handler {
		return controllers.RequireActor(errs, next)
	}

	controllers.NewLotController(createLotAdapter, errs, log).Register(mux, requireActor)
}

func repositories(pool *pgxpool.Pool) (ports.ActorRepository, ports.FarmRepository, ports.LotRepository) {
	if pool == nil {
		return memory.NewActorRepository(seedActors()...),
			memory.NewFarmRepository(seedFarms()...),
			memory.NewLotRepository()
	}

	return postgres.NewActorRepository(pool),
		postgres.NewFarmRepository(pool),
		postgres.NewLotRepository(pool)
}
