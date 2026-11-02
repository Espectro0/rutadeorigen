package main

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/lib/catalog"
	completeevidenceadapter "rutadeorigen/traceability/internal/application/input/adapters/completeevidenceupload"
	createfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/createfarm"
	createlotadapter "rutadeorigen/traceability/internal/application/input/adapters/createlot"
	deletefarmadapter "rutadeorigen/traceability/internal/application/input/adapters/deletefarm"
	getlotadapter "rutadeorigen/traceability/internal/application/input/adapters/getlot"
	getmeadapter "rutadeorigen/traceability/internal/application/input/adapters/getme"
	listactorsadapter "rutadeorigen/traceability/internal/application/input/adapters/listactors"
	listevidencesadapter "rutadeorigen/traceability/internal/application/input/adapters/listevidences"
	listfarmadapter "rutadeorigen/traceability/internal/application/input/adapters/listfarm"
	listlotseventsadapter "rutadeorigen/traceability/internal/application/input/adapters/listlotevents"
	listlotsadapter "rutadeorigen/traceability/internal/application/input/adapters/listlots"
	recordevidenceadapter "rutadeorigen/traceability/internal/application/input/adapters/recordevidenceresult"
	registeractoradapter "rutadeorigen/traceability/internal/application/input/adapters/registeractor"
	registerstageadapter "rutadeorigen/traceability/internal/application/input/adapters/registerstage"
	requestevidenceadapter "rutadeorigen/traceability/internal/application/input/adapters/requestevidenceupload"
	resolveactoradapter "rutadeorigen/traceability/internal/application/input/adapters/resolveactor"
	transferlotadapter "rutadeorigen/traceability/internal/application/input/adapters/transferlot"
	voidlotadapter "rutadeorigen/traceability/internal/application/input/adapters/voidlot"
	"rutadeorigen/traceability/internal/application/output/ports"
	completeevidenceuc "rutadeorigen/traceability/internal/application/usecases/completeevidenceupload"
	createfarmuc "rutadeorigen/traceability/internal/application/usecases/createfarm"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	deletefarmuc "rutadeorigen/traceability/internal/application/usecases/deletefarm"
	getlotuc "rutadeorigen/traceability/internal/application/usecases/getlot"
	listactorsuc "rutadeorigen/traceability/internal/application/usecases/listactors"
	listevidencesuc "rutadeorigen/traceability/internal/application/usecases/listevidences"
	listfarmuc "rutadeorigen/traceability/internal/application/usecases/listfarm"
	listlotseventsuc "rutadeorigen/traceability/internal/application/usecases/listlotevents"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	recordevidenceuc "rutadeorigen/traceability/internal/application/usecases/recordevidenceresult"
	registeractoruc "rutadeorigen/traceability/internal/application/usecases/registeractor"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
	requestevidenceuc "rutadeorigen/traceability/internal/application/usecases/requestevidenceupload"
	resolveactoruc "rutadeorigen/traceability/internal/application/usecases/resolveactor"
	transferlotuc "rutadeorigen/traceability/internal/application/usecases/transferlot"
	voidlotuc "rutadeorigen/traceability/internal/application/usecases/voidlot"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	inputmessaging "rutadeorigen/traceability/internal/infrastructure/input/messaging"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
	evidencestorage "rutadeorigen/traceability/internal/infrastructure/output/external/storage"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"
)

type routeConfig struct {
	storage  ports.EvidenceStorage
	settings requestevidenceuc.Settings
}

type routeOption func(*routeConfig)

func withEvidenceStorage(storage ports.EvidenceStorage) routeOption {
	return func(c *routeConfig) { c.storage = storage }
}

func withEvidenceSettings(settings requestevidenceuc.Settings) routeOption {
	return func(c *routeConfig) { c.settings = settings }
}

// registerRoutes returns the handler for the evidence results consumed from the broker.
func registerRoutes(mux *http.ServeMux, log *slog.Logger, pool *pgxpool.Pool, catalog *catalog.Catalog, authn authn,
	opts ...routeOption) *inputmessaging.EvidenceResultsHandler {
	cfg := routeConfig{
		storage:  evidencestorage.NewMemory(),
		settings: requestevidenceuc.Settings{MaxBytes: defaultEvidenceMaxBytes, UploadTTL: defaultEvidenceUploadTTL},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	actors, farms, lots, events := repositories(pool)
	evidences := evidenceRepository(pool)
	auth := authorizer.MustNewCasbin()
	tx, publisher := outputs(pool, log)
	errs := controllers.NewErrorResponder(log, catalog)

	createLotUseCase := createlotuc.New(actors, farms, lots, auth, publisher, tx)
	createLotAdapter := createlotadapter.New(createLotUseCase)
	getLotAdapter := getlotadapter.New(getlotuc.New(actors, lots, auth))
	listLotsAdapter := listlotsadapter.New(listlotsuc.New(actors, lots, auth))
	createFarmAdapter := createfarmadapter.New(createfarmuc.New(actors, farms, auth))
	listFarmsAdapter := listfarmadapter.New(listfarmuc.New(actors, farms, auth))
	registerStageAdapter := registerstageadapter.New(registerstageuc.New(actors, lots, events, auth, publisher, tx))
	transferLotAdapter := transferlotadapter.New(transferlotuc.New(actors, lots, events, auth, publisher, tx))
	listActorsAdapter := listactorsadapter.New(listactorsuc.New(actors, auth))
	listLotEventsAdapter := listlotseventsadapter.New(listlotseventsuc.New(actors, lots, events, auth))
	voidLotAdapter := voidlotadapter.New(voidlotuc.New(actors, lots, auth, publisher, tx))
	deleteFarmAdapter := deletefarmadapter.New(deletefarmuc.New(actors, farms, lots, auth, publisher))
	requestEvidenceAdapter := requestevidenceadapter.New(requestevidenceuc.New(actors, lots, evidences, cfg.storage, auth, publisher, cfg.settings))
	completeEvidenceAdapter := completeevidenceadapter.New(completeevidenceuc.New(actors, lots, evidences, cfg.storage, publisher, tx))
	listEvidencesAdapter := listevidencesadapter.New(listevidencesuc.New(actors, lots, evidences, auth))
	recordEvidenceAdapter := recordevidenceadapter.New(recordevidenceuc.New(evidences))

	resolver := resolveactoruc.New(actors)
	requireActor, authenticated := authn.middlewares(errs, resolveactoradapter.New(resolver))

	controllers.NewLotController(createLotAdapter, getLotAdapter, listLotsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewFarmController(createFarmAdapter, listFarmsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewLotEventController(registerStageAdapter, errs, log).Register(mux, requireActor)
	controllers.NewLotTransferController(transferLotAdapter, errs, log).Register(mux, requireActor)
	controllers.NewActorController(listActorsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewLotHistoryController(listLotEventsAdapter, errs, log).Register(mux, requireActor)
	controllers.NewLotVoidController(voidLotAdapter, errs, log).Register(mux, requireActor)
	controllers.NewFarmDeleteController(deleteFarmAdapter, errs, log).Register(mux, requireActor)
	controllers.NewEvidenceController(requestEvidenceAdapter, completeEvidenceAdapter, listEvidencesAdapter, errs, log).Register(mux, requireActor)

	if authenticated != nil {
		registerActorAdapter := registeractoradapter.New(registeractoruc.New(actors, publisher, tx))
		meAdapter := getmeadapter.New(resolver)
		controllers.NewMeController(registerActorAdapter, meAdapter, errs, log).Register(mux, authenticated)
		authn.routes.Register(mux)
	}

	return inputmessaging.NewEvidenceResultsHandler(recordEvidenceAdapter)
}

func evidenceRepository(pool *pgxpool.Pool) ports.EvidenceRepository {
	if pool == nil {
		return memory.NewEvidenceRepository()
	}
	return postgres.NewEvidenceRepository(pool)
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
