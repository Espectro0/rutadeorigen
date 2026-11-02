package ports

import "rutadeorigen/processor/internal/domain/rejection"

type RejectionMessages interface {
	Message(reason rejection.Reason) string
}
