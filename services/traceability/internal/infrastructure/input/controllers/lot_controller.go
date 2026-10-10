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

var errInvalidBody = exception.NewValidation(exception.LayerInfrastructure,
	"El cuerpo de la petición no es un JSON válido", "cannot decode request body")

type Middleware func(http.Handler) http.Handler

type LotController struct {
	createLot ports.CreateLotPort
	log       *slog.Logger
}

func NewLotController(createLot ports.CreateLotPort, log *slog.Logger) *LotController {
	return &LotController{createLot: createLot, log: log}
}

func (c *LotController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots", requireActor(http.HandlerFunc(c.create)))
}

func (c *LotController) create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateLot

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, c.log, errInvalidBody)
		return
	}

	res, err := c.createLot.Execute(r.Context(), req)
	if err != nil {
		writeError(w, r, c.log, err)
		return
	}

	w.Header().Set("Location", "/lots/"+res.ID)
	writeJSON(w, c.log, http.StatusCreated, res)
}
