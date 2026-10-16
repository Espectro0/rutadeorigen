package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

const maxBodyBytes = 1 << 20

type Middleware func(http.Handler) http.Handler

type LotController struct {
	createLot ports.CreateLotPort
	errs      *ErrorResponder
	log       *slog.Logger
}

func NewLotController(createLot ports.CreateLotPort, errs *ErrorResponder, log *slog.Logger) *LotController {
	return &LotController{createLot: createLot, errs: errs, log: log}
}

func (c *LotController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots", requireActor(http.HandlerFunc(c.create)))
}

func (c *LotController) create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateLot

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}

	res, err := c.createLot.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Location", "/lots/"+res.ID)
	writeJSON(w, c.log, http.StatusCreated, res)
}
