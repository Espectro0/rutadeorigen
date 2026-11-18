package main

import (
	"github.com/jackc/pgx/v5/pgxpool"

	sendadapter "rutadeorigen/notifications/internal/application/input/adapters/sendevidencenotification"
	"rutadeorigen/notifications/internal/application/output/ports"
	senduc "rutadeorigen/notifications/internal/application/usecases/sendevidencenotification"
	inputmessaging "rutadeorigen/notifications/internal/infrastructure/input/messaging"
	"rutadeorigen/notifications/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/notifications/internal/infrastructure/output/repositories/structured/postgres"
)

type persisted struct {
	processed ports.ProcessedEventRepository
	log       ports.NotificationLogRepository
	tx        ports.TxManager
}

func newEvidenceResultHandler(pool *pgxpool.Pool, renderer ports.EmailRenderer,
	sender ports.EmailSender) *inputmessaging.EvidenceResultHandler {
	store := persistence(pool)

	useCase := senduc.New(store.processed, store.log, renderer, sender, store.tx)

	return inputmessaging.NewEvidenceResultHandler(sendadapter.New(useCase))
}

func persistence(pool *pgxpool.Pool) persisted {
	if pool == nil {
		return persisted{
			processed: memory.NewProcessedEventRepository(),
			log:       memory.NewNotificationLogRepository(),
			tx:        memory.NewTxManager(),
		}
	}
	return persisted{
		processed: postgres.NewProcessedEventRepository(pool),
		log:       postgres.NewNotificationLogRepository(pool),
		tx:        postgres.NewTxManager(pool),
	}
}
