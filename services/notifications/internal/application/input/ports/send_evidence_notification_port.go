package ports

import "rutadeorigen/notifications/internal/application/input/dto/request"

type SendEvidenceNotificationPort interface {
	InputPortWithoutReturn[request.SendEvidenceNotification]
}
