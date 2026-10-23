package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
)

type Middleware func(http.Handler) http.Handler

type LotController struct {
	getLot ports.GetPublicLotPort
	errs   *ErrorResponder
	log    *slog.Logger
}

func NewLotController(getLot ports.GetPublicLotPort, errs *ErrorResponder, log *slog.Logger) *LotController {
	return &LotController{getLot: getLot, errs: errs, log: log}
}

func (c *LotController) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /public/lots/{id}", c.get)
}

func (c *LotController) get(w http.ResponseWriter, r *http.Request) {
	res, err := c.getLot.Execute(r.Context(), request.GetPublicLot{ID: r.PathValue("id")})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, c.log, http.StatusOK, res)
}
