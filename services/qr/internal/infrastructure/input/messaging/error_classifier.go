package messaging

import (
	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/qr/internal/platform/exception"
)

func classify(err error) error {
	if err == nil {
		return nil
	}

	if ex, ok := exception.From(err); ok && ex.Kind() == exception.KindValidation {
		return libmessaging.Permanent(err)
	}

	return err
}
