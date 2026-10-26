package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

type LotEventController struct {
	registerStage ports.RegisterStagePort
	errs          *ErrorResponder
	log           *slog.Logger
}

func NewLotEventController(registerStage ports.RegisterStagePort, errs *ErrorResponder, log *slog.Logger) *LotEventController {
	return &LotEventController{registerStage: registerStage, errs: errs, log: log}
}

func (c *LotEventController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots/{id}/events", requireActor(http.HandlerFunc(c.registerStageEvent)))
}

func (c *LotEventController) registerStageEvent(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterStage

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}
	req.LotID = r.PathValue("id")
	req.IfMatch = r.Header.Get("If-Match")

	res, err := c.registerStage.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(res.LotVersion)))
	writeJSON(w, c.log, http.StatusCreated, res)
}
