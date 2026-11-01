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

type LotTransferController struct {
	transferLot ports.TransferLotPort
	errs        *ErrorResponder
	log         *slog.Logger
}

func NewLotTransferController(transferLot ports.TransferLotPort, errs *ErrorResponder, log *slog.Logger) *LotTransferController {
	return &LotTransferController{transferLot: transferLot, errs: errs, log: log}
}

func (c *LotTransferController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("POST /lots/{id}/transfers", requireActor(http.HandlerFunc(c.transfer)))
}

func (c *LotTransferController) transfer(w http.ResponseWriter, r *http.Request) {
	var req request.TransferLot

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		c.errs.Write(w, r, exception.New(exception.CodeInvalidRequestBody))
		return
	}
	req.LotID = r.PathValue("id")
	req.IfMatch = r.Header.Get("If-Match")

	res, err := c.transferLot.Execute(r.Context(), req)
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(res.LotVersion)))
	writeJSON(w, c.log, http.StatusCreated, res)
}
