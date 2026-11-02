package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

type EvidenceController struct {
	requestUpload  ports.RequestEvidenceUploadPort
	completeUpload ports.CompleteEvidenceUploadPort
	list           ports.ListEvidencesPort
	errs           *ErrorResponder
	log            *slog.Logger
}

func NewEvidenceController(requestUpload ports.RequestEvidenceUploadPort, completeUpload ports.CompleteEvidenceUploadPort,
	list ports.ListEvidencesPort, errs *ErrorResponder, log *slog.Logger) *EvidenceController {
	return &EvidenceController{requestUpload: requestUpload, completeUpload: completeUpload, list: list, errs: errs, log: log}
}

func (c *EvidenceController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots/{id}/evidences", requireActor(http.HandlerFunc(c.requestEvidenceUpload)))
	mux.Handle("POST /lots/{id}/evidences/{evidenceId}/complete", requireActor(http.HandlerFunc(c.completeEvidenceUpload)))
	mux.Handle("GET /lots/{id}/evidences", requireActor(http.HandlerFunc(c.listEvidences)))
}

func (c *EvidenceController) requestEvidenceUpload(w http.ResponseWriter, r *http.Request) {
	var req request.EvidenceUpload

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}
	req.LotID = r.PathValue("id")

	res, err := c.requestUpload.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusCreated, res)
}

func (c *EvidenceController) completeEvidenceUpload(w http.ResponseWriter, r *http.Request) {
	res, err := c.completeUpload.Execute(r.Context(), request.CompleteEvidenceUpload{
		LotID:      r.PathValue("id"),
		EvidenceID: r.PathValue("evidenceId"),
	})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}

func (c *EvidenceController) listEvidences(w http.ResponseWriter, r *http.Request) {
	res, err := c.list.Execute(r.Context(), request.ListEvidences{LotID: r.PathValue("id")})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
