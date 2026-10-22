package main

import (
	"log/slog"
	"net/http"
	createfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/createfarm"
	createlotadapter "rutadeorigen/traceability/internal/application/input/adapters/createlot"
	getlotadapter "rutadeorigen/traceability/internal/application/input/adapters/getlot"
	listfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/listfarm"
	createfarmuc "rutadeorigen/traceability/internal/application/usecases/createfarm"
	listfarmuc "rutadeorigen/traceability/internal/application/usecases/listfarm"

	"rutadeorigen/lib/catalog"
	listlotsadapter "rutadeorigen/traceability/internal/application/input/adapters/listlots"
	"rutadeorigen/traceability/internal/application/output/ports"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	getlotuc "rutadeorigen/traceability/internal/application/usecases/getlot"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerRoutes(mux *http.ServeMux, log *slog.Logger, pool *pgxpool.Pool, catalog *catalog.Catalog) {
	actors, farms, lots := repositories(pool)
	auth := authorizer.NewAllowAll()
	tx, publisher := outputs(pool, log)
	errs := controllers.NewErrorResponder(log, catalog)

	createLotUseCase := createlotuc.New(actors, farms, lots, auth, publisher, tx)
	createLotAdapter := createlotadapter.New(createLotUseCase)
	getLotAdapter := getlotadapter.New(getlotuc.New(actors, lots, auth))
	listLotsAdapter := listlotsadapter.New(listlotsuc.New(actors, lots, auth))
	createFarmAdapter := createfarmadapter.New(createfarmuc.New(actors, farms, auth))
	listFarmsAdapter := listfarmadapter.New(listfarmuc.New(actors, farms, auth))

	requireActor := func(next http.Handler) http.Handler {
		return controllers.RequireActor(errs, next)
	}

	controllers.NewLotController(createLotAdapter, getLotAdapter, listLotsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewFarmController(createFarmAdapter, listFarmsAdapter, errs, log).Register(mux, requireActor)
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

func outputs(pool *pgxpool.Pool, log *slog.Logger) (ports.TxManager, ports.EventPublisher) {
	if pool == nil {
		return memory.NewTxManager(), messaging.NewLogPublisher(log)
	}
	return postgres.NewTxManager(pool), postgres.NewOutboxPublisher(pool)
}
