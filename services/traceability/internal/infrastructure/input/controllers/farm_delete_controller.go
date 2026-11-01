package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
)

type FarmDeleteController struct {
	deleteFarm ports.DeleteFarmPort
	errs       *ErrorResponder
	log        *slog.Logger
}

func NewFarmDeleteController(deleteFarm ports.DeleteFarmPort, errs *ErrorResponder, log *slog.Logger) *FarmDeleteController {
	return &FarmDeleteController{deleteFarm: deleteFarm, errs: errs, log: log}
}

func (c *FarmDeleteController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("DELETE /farms/{id}", requireActor(http.HandlerFunc(c.delete)))
}

func (c *FarmDeleteController) delete(w http.ResponseWriter, r *http.Request) {
	if err := c.deleteFarm.Execute(r.Context(), request.DeleteFarm{ID: r.PathValue("id")}); err != nil {
		c.errs.Write(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
