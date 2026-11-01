package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
)

type ActorController struct {
	listActors ports.ListActorsPort
	errs       *ErrorResponder
	log        *slog.Logger
}

func NewActorController(listActors ports.ListActorsPort, errs *ErrorResponder, log *slog.Logger) *ActorController {
	return &ActorController{listActors: listActors, errs: errs, log: log}
}

func (c *ActorController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("GET /actors", requireActor(http.HandlerFunc(c.list)))
}

func (c *ActorController) list(w http.ResponseWriter, r *http.Request) {
	res, err := c.listActors.Execute(r.Context(), request.ListActors{Type: r.URL.Query().Get("type")})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
