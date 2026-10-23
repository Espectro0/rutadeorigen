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
	projectlotcreatedadapter "rutadeorigen/qr/internal/application/input/adapters/projectlotcreated"
	resolvescanadapter "rutadeorigen/qr/internal/application/input/adapters/resolvescan"
	"rutadeorigen/qr/internal/application/output/ports"
	getpubliclotuc "rutadeorigen/qr/internal/application/usecases/getpubliclot"
	getqrimageuc "rutadeorigen/qr/internal/application/usecases/getqrimage"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
	resolvescanuc "rutadeorigen/qr/internal/application/usecases/resolvescan"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	inmessaging "rutadeorigen/qr/internal/infrastructure/input/messaging"
	"rutadeorigen/qr/internal/infrastructure/output/external/authorizer"
	"rutadeorigen/qr/internal/infrastructure/output/external/renderer"
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
}

type outputs struct {
	views     ports.LotViewRepository
	codes     ports.QRCodeRepository
	rules     ports.RedirectRepository
	scans     ports.ScanRepository
	processed ports.ProcessedEventRepository
	cache     ports.LotViewCache
	tx        ports.TxManager
}

func registerRoutes(mux *http.ServeMux, log *slog.Logger, catalog *catalog.Catalog, deps dependencies) *inmessaging.LotCreatedHandler {
	out := newOutputs(deps)
	auth := authorizer.NewAllowAll()
	errs := controllers.NewErrorResponder(log, catalog)

	projectLotCreated := projectlotcreatedadapter.New(projectlotcreateduc.New(out.views, out.processed, out.cache, out.tx))
	getPublicLot := getpubliclotadapter.New(getpubliclotuc.New(out.views, out.cache))
	resolveScan := resolvescanadapter.New(resolvescanuc.New(out.rules, out.views, out.cache, out.scans))
	getQRImage := getqrimageadapter.New(getqrimageuc.New(out.views, out.cache, out.codes, renderer.NewGoQRCodeRenderer(), auth, deps.publicBaseURL))

	requireActor := func(next http.Handler) http.Handler {
		return controllers.RequireActor(errs, next)
	}

	controllers.NewLotController(getPublicLot, errs, log).Register(mux)
	controllers.NewScanController(resolveScan, errs).Register(mux)
	controllers.NewQRImageController(getQRImage, errs).Register(mux, requireActor)

	return inmessaging.NewLotCreatedHandler(projectLotCreated)
}

func newOutputs(deps dependencies) outputs {
	if deps.pool == nil || deps.redis == nil {
		return memoryOutputs()
	}

	return outputs{
		views:     postgres.NewLotViewRepository(deps.pool),
		codes:     postgres.NewQRCodeRepository(deps.pool),
		rules:     postgres.NewRedirectRepository(deps.pool),
		scans:     postgres.NewScanRepository(deps.pool),
		processed: postgres.NewProcessedEventRepository(deps.pool),
		cache:     semiredis.NewLotViewCache(deps.redis, deps.cacheTTL),
		tx:        postgres.NewTxManager(deps.pool),
	}
}

func memoryOutputs() outputs {
	return outputs{
		views:     memory.NewLotViewRepository(),
		codes:     memory.NewQRCodeRepository(),
		rules:     memory.NewRedirectRepository(seedRedirects()...),
		scans:     memory.NewScanRepository(),
		processed: memory.NewProcessedEventRepository(),
		cache:     semimemory.NewLotViewCache(),
		tx:        memory.NewTxManager(),
	}
}

func seedRedirects() []*redirect.Rule {
	rule, err := redirect.New("/l/", "/lotes/{id}", http.StatusMovedPermanently)
	if err != nil {
		return nil
	}
	return []*redirect.Rule{rule}
}
