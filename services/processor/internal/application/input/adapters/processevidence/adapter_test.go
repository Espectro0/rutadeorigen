package processevidence

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/processor/internal/application/input/dto/request"
	processuc "rutadeorigen/processor/internal/application/usecases/processevidence"
	"rutadeorigen/processor/internal/platform/exception"
)

type recordingUseCase struct {
	received []processuc.Command
	err      error
}

func (r *recordingUseCase) Execute(_ context.Context, cmd processuc.Command) error {
	r.received = append(r.received, cmd)
	return r.err
}

func validInput() request.ProcessEvidence {
	lotID, evidenceID := uuid.New(), uuid.New()
	return request.ProcessEvidence{
		MessageID:       uuid.NewString(),
		EventType:       " evidence.uploaded ",
		EvidenceID:      evidenceID.String(),
		LotID:           lotID.String(),
		Stage:           " farm ",
		QuarantineKey:   lotID.String() + "/" + evidenceID.String(),
		SizeBytes:       2048,
		UploaderActorID: uuid.NewString(),
		UploaderRole:    " producer ",
	}
}

func TestExecuteBuildsCommand(t *testing.T) {
	useCase := &recordingUseCase{}
	input := validInput()

	if err := New(useCase).Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}

	if len(useCase.received) != 1 {
		t.Fatalf("want one call, got %d", len(useCase.received))
	}
	cmd := useCase.received[0]
	if cmd.EventType != "evidence.uploaded" || cmd.Uploader.Role != "producer" || cmd.Stage != "farm" {
		t.Errorf("fields must be trimmed: %+v", cmd)
	}
	if cmd.EvidenceID.String() != input.EvidenceID || cmd.SizeBytes != 2048 {
		t.Errorf("unexpected command %+v", cmd)
	}
}

func TestExecuteRejectsInvalidIdentifiers(t *testing.T) {
	cases := map[string]func(in *request.ProcessEvidence){
		"message id":  func(in *request.ProcessEvidence) { in.MessageID = "nope" },
		"evidence id": func(in *request.ProcessEvidence) { in.EvidenceID = "" },
		"lot id":      func(in *request.ProcessEvidence) { in.LotID = "x" },
		"uploader id": func(in *request.ProcessEvidence) { in.UploaderActorID = "" },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			useCase := &recordingUseCase{}
			input := validInput()
			breakIt(&input)

			err := New(useCase).Execute(context.Background(), input)
			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("want invalid payload, got %v", err)
			}
			if len(useCase.received) != 0 {
				t.Error("use case must not be called")
			}
		})
	}
}

func TestExecutePropagatesUseCaseError(t *testing.T) {
	failure := errors.New("boom")

	err := New(&recordingUseCase{err: failure}).Execute(context.Background(), validInput())

	if !errors.Is(err, failure) {
		t.Errorf("want use case error, got %v", err)
	}
}
