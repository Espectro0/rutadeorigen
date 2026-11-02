package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"rutadeorigen/lib/catalog"
	getpubliclotadapter "rutadeorigen/qr/internal/application/input/adapters/getpubliclot"
	getqrimageadapter "rutadeorigen/qr/internal/application/input/adapters/getqrimage"
	getscanstatsadapter "rutadeorigen/qr/internal/application/input/adapters/getscanstats"
	projectactorregisteredadapter "rutadeorigen/qr/internal/application/input/adapters/projectactorregistered"
	projectevidenceprocessedadapter "rutadeorigen/qr/internal/application/input/adapters/projectevidenceprocessed"
	projectlotcreatedadapter "rutadeorigen/qr/internal/application/input/adapters/projectlotcreated"
	projectlotstageadapter "rutadeorigen/qr/internal/application/input/adapters/projectlotstage"
	projectlottransferredadapter "rutadeorigen/qr/internal/application/input/adapters/projectlottransferred"
	projectlotvoidedadapter "rutadeorigen/qr/internal/application/input/adapters/projectlotvoided"
	resolveactoradapter "rutadeorigen/qr/internal/application/input/adapters/resolveactor"
	resolvescanadapter "rutadeorigen/qr/internal/application/input/adapters/resolvescan"
	"rutadeorigen/qr/internal/application/output/ports"
	getpubliclotuc "rutadeorigen/qr/internal/application/usecases/getpubliclot"
	getqrimageuc "rutadeorigen/qr/internal/application/usecases/getqrimage"
	getscanstatsuc "rutadeorigen/qr/internal/application/usecases/getscanstats"
	listpublicevidencesuc "rutadeorigen/qr/internal/application/usecases/listpublicevidences"
	projectactorregistereduc "rutadeorigen/qr/internal/application/usecases/projectactorregistered"
	projectevidenceprocesseduc "rutadeorigen/qr/internal/application/usecases/projectevidenceprocessed"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
	projectlotstageuc "rutadeorigen/qr/internal/application/usecases/projectlotstage"
	projectlottransferreduc "rutadeorigen/qr/internal/application/usecases/projectlottransferred"
	projectlotvoideduc "rutadeorigen/qr/internal/application/usecases/projectlotvoided"
	resolveactoruc "rutadeorigen/qr/internal/application/usecases/resolveactor"
	resolvescanuc "rutadeorigen/qr/internal/application/usecases/resolvescan"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	inmessaging "rutadeorigen/qr/internal/infrastructure/input/messaging"
	"rutadeorigen/qr/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/qr/internal/infrastructure/output/external/renderer"
	evidencestorage "rutadeorigen/qr/internal/infrastructure/output/external/storage"
	semimemory "rutadeorigen/qr/internal/infrastructure/output/repositories/semistructured/memory"
	semiredis "rutadeorigen/qr/internal/infrastructure/output/repositories/semistructured/redis"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres"
)

type dependencies struct {
	pool          *pgxpool.Pool
	redis         *goredis.Client
	publicBaseURL string
	cacheTTL      time.Duration
	viewer        ports.EvidenceViewer
	evidence      listpublicevidencesuc.Settings
}

type outputs struct {
	views     ports.LotViewRepository
	actors    ports.ActorViewRepository
	codes     ports.QRCodeRepository
	rules     ports.RedirectRepository
	scans     ports.ScanRepository
	processed ports.ProcessedEventRepository
	evidences ports.EvidenceViewRepository
	viewer    ports.EvidenceViewer
	cache     ports.LotViewCache
	tx        ports.TxManager
}

func registerRoutes(mux *http.ServeMux, log *slog.Logger, catalog *catalog.Catalog, deps dependencies, authn authn) *inmessaging.LotEventsHandler {
	out := newOutputs(deps)
	auth := authorizer.MustNewCasbin()
	errs := controllers.NewErrorResponder(log, catalog)

	projectLotCreated := projectlotcreatedadapter.New(projectlotcreateduc.New(out.views, out.processed, out.cache, out.tx))
	projectLotStaged := projectlotstageadapter.New(projectlotstageuc.New(out.views, out.processed, out.cache, out.tx))
	projectActorRegistered := projectactorregisteredadapter.New(projectactorregistereduc.New(out.actors, out.processed, out.tx))
	projectLotTransferred := projectlottransferredadapter.New(projectlottransferreduc.New(out.views, out.processed, out.cache, out.tx))
	projectLotVoided := projectlotvoidedadapter.New(projectlotvoideduc.New(out.views, out.processed, out.cache, out.tx))
	projectEvidenceProcessed := projectevidenceprocessedadapter.New(projectevidenceprocesseduc.New(out.evidences, out.processed, out.tx))
	getPublicLot := getpubliclotadapter.New(getpubliclotuc.New(out.views, out.cache)).
		WithEvidences(listpublicevidencesuc.New(out.evidences, out.viewer, evidenceSettings(deps.evidence)))
	resolveScan := resolvescanadapter.New(resolvescanuc.New(out.rules, out.views, out.cache, out.scans))
	getQRImage := getqrimageadapter.New(getqrimageuc.New(out.views, out.cache, out.codes, renderer.NewGoQRCodeRenderer(), auth, deps.publicBaseURL))

	getScanStats := getscanstatsadapter.New(getscanstatsuc.New(out.views, out.cache, out.scans, auth))
	authn.resolver = resolveactoradapter.New(resolveactoruc.New(out.actors))
	requireActor := authn.requireActor(errs)

	controllers.NewLotController(getPublicLot, errs, log).Register(mux)
	controllers.NewScanController(resolveScan, errs).Register(mux)
	controllers.NewQRImageController(getQRImage, errs).Register(mux, requireActor)
	controllers.NewScanStatsController(getScanStats, errs, log).Register(mux, requireActor)

	return inmessaging.NewLotEventsHandler(projectLotCreated, projectLotStaged).
		WithActorRegistered(projectActorRegistered).
		WithLotTransferred(projectLotTransferred).
		WithLotVoided(projectLotVoided).
		WithEvidenceProcessed(projectEvidenceProcessed)
}

func evidenceSettings(configured listpublicevidencesuc.Settings) listpublicevidencesuc.Settings {
	settings := defaultEvidenceSettings()
	if configured.ViewTTL > 0 {
		settings.ViewTTL = configured.ViewTTL
	}
	if configured.MaxPerLot > 0 {
		settings.MaxPerLot = configured.MaxPerLot
	}
	return settings
}

func evidenceViewer(configured ports.EvidenceViewer) ports.EvidenceViewer {
	if configured == nil {
		return evidencestorage.NewMemory()
	}
	return configured
}

func newOutputs(deps dependencies) outputs {
	if deps.pool == nil || deps.redis == nil {
		return memoryOutputs()
	}

	return outputs{
		views:     postgres.NewLotViewRepository(deps.pool),
		actors:    postgres.NewActorViewRepository(deps.pool),
		codes:     postgres.NewQRCodeRepository(deps.pool),
		rules:     postgres.NewRedirectRepository(deps.pool),
		scans:     postgres.NewScanRepository(deps.pool),
		processed: postgres.NewProcessedEventRepository(deps.pool),
		evidences: postgres.NewEvidenceViewRepository(deps.pool),
		viewer:    evidenceViewer(deps.viewer),
		cache:     semiredis.NewLotViewCache(deps.redis, deps.cacheTTL),
		tx:        postgres.NewTxManager(deps.pool),
	}
}

func memoryOutputs() outputs {
	return outputs{
		views:     memory.NewLotViewRepository(),
		actors:    memory.NewActorViewRepository(),
		codes:     memory.NewQRCodeRepository(),
		rules:     memory.NewRedirectRepository(seedRedirects()...),
		scans:     memory.NewScanRepository(),
		processed: memory.NewProcessedEventRepository(),
		evidences: memory.NewEvidenceViewRepository(),
		viewer:    evidencestorage.NewMemory(),
		cache:     semimemory.NewLotViewCache(),
		tx:        memory.NewTxManager(),
	}
}

func seedRedirects() []*redirect.Rule {
	rule, err := redirect.New("/l/", "/lotes/{id}", http.StatusFound)
	if err != nil {
		return nil
	}
	return []*redirect.Rule{rule}
}
