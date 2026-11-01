package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
)

type ScanStatsController struct {
	getStats ports.GetScanStatsPort
	errs     *ErrorResponder
	log      *slog.Logger
}

func NewScanStatsController(getStats ports.GetScanStatsPort, errs *ErrorResponder, log *slog.Logger) *ScanStatsController {
	return &ScanStatsController{getStats: getStats, errs: errs, log: log}
}

func (c *ScanStatsController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("GET /lots/{id}/scans/stats", requireActor(http.HandlerFunc(c.get)))
}

func (c *ScanStatsController) get(w http.ResponseWriter, r *http.Request) {
	res, err := c.getStats.Execute(r.Context(), request.GetScanStats{
		LotID: r.PathValue("id"),
		Days:  r.URL.Query().Get("days"),
	})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, c.log, http.StatusOK, res)
}
