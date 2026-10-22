package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

type FarmController struct {
	createFarm ports.CreateFarmPort
	listFarms  ports.ListFarmsPort
	errs       *ErrorResponder
	log        *slog.Logger
}

func NewFarmController(createFarm ports.CreateFarmPort, listFarms ports.ListFarmsPort, errs *ErrorResponder, log *slog.Logger) *FarmController {
	return &FarmController{createFarm: createFarm, listFarms: listFarms, errs: errs, log: log}
}

func (c *FarmController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /farms", requireActor(http.HandlerFunc(c.create)))
	mux.Handle("GET /farms", requireActor(http.HandlerFunc(c.list)))
}

func (c *FarmController) create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateFarm

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}

	res, err := c.createFarm.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Location", "/farms/"+res.ID)
	writeJSON(w, c.log, http.StatusCreated, res)
}

func (c *FarmController) list(w http.ResponseWriter, r *http.Request) {
	res, err := c.listFarms.Execute(r.Context(), request.ListFarms{})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
