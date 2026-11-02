package rejection

import (
	"errors"
	"testing"

	"rutadeorigen/processor/internal/platform/exception"
)

func TestFromError(t *testing.T) {
	r := FromError(exception.New(exception.CodeImageCorrupt))
	if r.IsDefault() || r.Code() != string(exception.CodeImageCorrupt) {
		t.Errorf("got %v", r)
	}

	for _, err := range []error{
		nil,
		errors.New("boom"),
		exception.New(exception.CodeStorageUnavailable),
		exception.New(exception.CodeEventInvalidPayload),
		exception.New(exception.CodeImageProcessingFailed),
	} {
		if !FromError(err).IsDefault() {
			t.Errorf("%v must not be a rejection", err)
		}
	}
}

func TestParamsAreCopied(t *testing.T) {
	r := FromError(exception.New(exception.CodeFileTooLarge).WithParams(map[string]any{"max": 5}))
	p := r.Params()
	p["max"] = 99
	if r.Params()["max"] != 5 {
		t.Error("params must be cloned")
	}
}
