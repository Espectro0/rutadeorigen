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

const maxBodyBytes = 1 << 20

type Middleware func(http.Handler) http.Handler

type LotController struct {
	createLot ports.CreateLotPort
	getLot    ports.GetLotPort
	listLots  ports.ListLotsPort
	errs      *ErrorResponder
	log       *slog.Logger
}

func NewLotController(createLot ports.CreateLotPort, getLot ports.GetLotPort, listLots ports.ListLotsPort, errs *ErrorResponder, log *slog.Logger) *LotController {
	return &LotController{createLot: createLot, getLot: getLot, listLots: listLots, errs: errs, log: log}
}

func (c *LotController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots", requireActor(http.HandlerFunc(c.create)))
	mux.Handle("GET /lots/{id}", requireActor(http.HandlerFunc(c.get)))
	mux.Handle("GET /lots", requireActor(http.HandlerFunc(c.list)))
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

func (c *LotController) get(w http.ResponseWriter, r *http.Request) {
	res, err := c.getLot.Execute(r.Context(), request.GetLot{ID: r.PathValue("id")})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Etag", strconv.Quote(strconv.Itoa(res.Version)))
	writeJSON(w, c.log, http.StatusOK, res)
}

func (c *LotController) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := c.listLots.Execute(r.Context(), request.ListLots{
		FarmID:      q.Get("farm_id"),
		CustodianID: q.Get("custodian_id"),
		Cursor:      q.Get("cursor"),
		Limit:       q.Get("limit"),
	})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}
	writeJSON(w, c.log, http.StatusOK, res)
}
