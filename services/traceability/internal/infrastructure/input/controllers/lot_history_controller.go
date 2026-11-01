package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
)

type LotHistoryController struct {
	listEvents ports.ListLotEventsPort
	errs       *ErrorResponder
	log        *slog.Logger
}

func NewLotHistoryController(listEvents ports.ListLotEventsPort, errs *ErrorResponder, log *slog.Logger) *LotHistoryController {
	return &LotHistoryController{listEvents: listEvents, errs: errs, log: log}
}

func (c *LotHistoryController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("GET /lots/{id}/events", requireActor(http.HandlerFunc(c.list)))
}

func (c *LotHistoryController) list(w http.ResponseWriter, r *http.Request) {
	res, err := c.listEvents.Execute(r.Context(), request.ListLotEvents{LotID: r.PathValue("id")})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
