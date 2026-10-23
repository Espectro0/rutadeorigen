package controllers

import (
	"net/http"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
)

type ScanController struct {
	resolveScan ports.ResolveScanPort
	errs        *ErrorResponder
}

func NewScanController(resolveScan ports.ResolveScanPort, errs *ErrorResponder) *ScanController {
	return &ScanController{resolveScan: resolveScan, errs: errs}
}

func (c *ScanController) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /l/{id}", c.scan)
}

func (c *ScanController) scan(w http.ResponseWriter, r *http.Request) {
	res, err := c.resolveScan.Execute(r.Context(), request.ResolveScan{Path: r.URL.Path})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Location", res.Location)
	w.WriteHeader(res.StatusCode)
}
