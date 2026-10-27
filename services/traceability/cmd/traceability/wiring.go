package main

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/lib/catalog"
	createfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/createfarm"
	createlotadapter "rutadeorigen/traceability/internal/application/input/adapters/createlot"
	getlotadapter "rutadeorigen/traceability/internal/application/input/adapters/getlot"
	getmeadapter "rutadeorigen/traceability/internal/application/input/adapters/getme"
	listfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/listfarm"
	listlotsadapter "rutadeorigen/traceability/internal/application/input/adapters/listlots"
	registeractoradapter "rutadeorigen/traceability/internal/application/input/adapters/registeractor"
	registerstageadapter "rutadeorigen/traceability/internal/application/input/adapters/registerstage"
	"rutadeorigen/traceability/internal/application/output/ports"
	createfarmuc "rutadeorigen/traceability/internal/application/usecases/createfarm"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	getlotuc "rutadeorigen/traceability/internal/application/usecases/getlot"
	listfarmuc "rutadeorigen/traceability/internal/application/usecases/listfarm"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	registeractoruc "rutadeorigen/traceability/internal/application/usecases/registeractor"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
	resolveactoruc "rutadeorigen/traceability/internal/application/usecases/resolveactor"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"
)

func registerRoutes(mux *http.ServeMux, log *slog.Logger, pool *pgxpool.Pool, catalog *catalog.Catalog, authn authn) {
	actors, farms, lots, events := repositories(pool)
	auth := authorizer.NewAllowAll()
	tx, publisher := outputs(pool, log)
	errs := controllers.NewErrorResponder(log, catalog)

	createLotUseCase := createlotuc.New(actors, farms, lots, auth, publisher, tx)
	createLotAdapter := createlotadapter.New(createLotUseCase)
	getLotAdapter := getlotadapter.New(getlotuc.New(actors, lots, auth))
	listLotsAdapter := listlotsadapter.New(listlotsuc.New(actors, lots, auth))
	createFarmAdapter := createfarmadapter.New(createfarmuc.New(actors, farms, auth))
	listFarmsAdapter := listfarmadapter.New(listfarmuc.New(actors, farms, auth))
	registerStageAdapter := registerstageadapter.New(registerstageuc.New(actors, lots, events, auth, publisher, tx))

	resolver := resolveactoruc.New(actors)
	requireActor, authenticated := authn.middlewares(errs, resolver)

	controllers.NewLotController(createLotAdapter, getLotAdapter, listLotsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewFarmController(createFarmAdapter, listFarmsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewLotEventController(registerStageAdapter, errs, log).Register(mux, requireActor)

	if authenticated != nil {
		registerActorAdapter := registeractoradapter.New(registeractoruc.New(actors))
		meAdapter := getmeadapter.New(resolver)
		controllers.NewMeController(registerActorAdapter, meAdapter, errs, log).Register(mux, authenticated)
		authn.routes.Register(mux)
	}
}

func repositories(pool *pgxpool.Pool) (ports.ActorRepository, ports.FarmRepository, ports.LotRepository, ports.LotEventRepository) {
	if pool == nil {
		return memory.NewActorRepository(seedActors()...),
			memory.NewFarmRepository(seedFarms()...),
			memory.NewLotRepository(),
			memory.NewLotEventRepository()
	}

	return postgres.NewActorRepository(pool),
		postgres.NewFarmRepository(pool),
		postgres.NewLotRepository(pool),
		postgres.NewLotEventRepository(pool)
}

func outputs(pool *pgxpool.Pool, log *slog.Logger) (ports.TxManager, ports.EventPublisher) {
	if pool == nil {
		return memory.NewTxManager(), messaging.NewLogPublisher(log)
	}
	return postgres.NewTxManager(pool), postgres.NewOutboxPublisher(pool)
}
