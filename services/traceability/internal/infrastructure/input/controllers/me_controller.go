package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

type MeController struct {
	registerActor ports.RegisterActorPort
	getMe         ports.GetMePort
	errs          *ErrorResponder
	log           *slog.Logger
}

func NewMeController(registerActor ports.RegisterActorPort, getMe ports.GetMePort, errs *ErrorResponder, log *slog.Logger) *MeController {
	return &MeController{registerActor: registerActor, getMe: getMe, errs: errs, log: log}
}

func (c *MeController) Register(mux *http.ServeMux, authenticated Middleware) {
	mux.Handle("POST /me", authenticated(http.HandlerFunc(c.register)))
	mux.Handle("GET /me", authenticated(http.HandlerFunc(c.get)))
}

func (c *MeController) register(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterActor
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}

	res, err := c.registerActor.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	status := http.StatusOK
	if res.Created {
		status = http.StatusCreated
		w.Header().Set("Location", "/me")
	}
	writeJSON(w, c.log, status, res.Actor)
}

func (c *MeController) get(w http.ResponseWriter, r *http.Request) {
	res, err := c.getMe.Execute(r.Context(), request.GetMe{})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
