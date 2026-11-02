package main

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	processadapter "rutadeorigen/processor/internal/application/input/adapters/processevidence"
	"rutadeorigen/processor/internal/application/output/ports"
	processuc "rutadeorigen/processor/internal/application/usecases/processevidence"
	"rutadeorigen/processor/internal/domain/picture"
	inputmessaging "rutadeorigen/processor/internal/infrastructure/input/messaging"
	"rutadeorigen/processor/internal/infrastructure/output/external/imaging"
	"rutadeorigen/processor/internal/infrastructure/output/messaging"
	"rutadeorigen/processor/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/processor/internal/infrastructure/output/repositories/structured/postgres"
)

type buckets struct {
	quarantine ports.QuarantineStorage
	evidences  ports.EvidenceStorage
}

func newEvidenceUploadedHandler(log *slog.Logger, pool *pgxpool.Pool, stores buckets,
	messages ports.RejectionMessages, limits picture.Limits) *inputmessaging.EvidenceUploadedHandler {
	processed, tx, publisher := persistence(pool, log)

	useCase := processuc.New(processed, stores.quarantine, stores.evidences, imaging.New(),
		messages, publisher, tx, limits)

	return inputmessaging.NewEvidenceUploadedHandler(processadapter.New(useCase))
}

func persistence(pool *pgxpool.Pool, log *slog.Logger) (ports.ProcessedEventRepository, ports.TxManager, ports.EventPublisher) {
	if pool == nil {
		return memory.NewProcessedEventRepository(), memory.NewTxManager(), messaging.NewLogPublisher(log)
	}
	return postgres.NewProcessedEventRepository(pool), postgres.NewTxManager(pool), postgres.NewOutboxPublisher(pool)
}
