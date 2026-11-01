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

type LotVoidController struct {
	voidLot ports.VoidLotPort
	errs    *ErrorResponder
	log     *slog.Logger
}

func NewLotVoidController(voidLot ports.VoidLotPort, errs *ErrorResponder, log *slog.Logger) *LotVoidController {
	return &LotVoidController{voidLot: voidLot, errs: errs, log: log}
}

func (c *LotVoidController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots/{id}/void", requireActor(http.HandlerFunc(c.void)))
}

func (c *LotVoidController) void(w http.ResponseWriter, r *http.Request) {
	var req request.VoidLot

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}
	req.LotID = r.PathValue("id")
	req.IfMatch = r.Header.Get("If-Match")

	res, err := c.voidLot.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(res.Version)))
	writeJSON(w, c.log, http.StatusOK, res)
}
