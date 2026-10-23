package controllers

import (
	"mime"
	"net/http"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
	"rutadeorigen/qr/internal/platform/helper"
)

type QRImageController struct {
	getImage ports.GetQRImagePort
	errs     *ErrorResponder
}

func NewQRImageController(getImage ports.GetQRImagePort, errs *ErrorResponder) *QRImageController {
	return &QRImageController{getImage: getImage, errs: errs}
}

func (c *QRImageController) Register(mux *http.ServeMux, requireActor Middleware) {
	mux.Handle("GET /public/lots/{id}/qr", requireActor(http.HandlerFunc(c.get)))
}

func (c *QRImageController) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := c.getImage.Execute(r.Context(), request.GetQRImage{
		LotID:  r.PathValue("id"),
		Size:   q.Get("size"),
		Format: q.Get("format"),
	})
	if err != nil {
		c.errs.Write(w, r, err)
		return
	}

	etag := `"` + helper.SHA256Hex(res.Data) + `"`
	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "private, max-age=300")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": res.Filename}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.Data)
}
