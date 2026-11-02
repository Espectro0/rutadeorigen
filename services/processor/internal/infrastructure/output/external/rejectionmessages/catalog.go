package rejectionmessages

import (
	libcatalog "rutadeorigen/lib/catalog"
	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/domain/rejection"
)

var _ ports.RejectionMessages = (*Catalog)(nil)

type Catalog struct {
	catalog *libcatalog.Catalog
}

func NewCatalog(catalog *libcatalog.Catalog) *Catalog {
	return &Catalog{catalog: catalog}
}

func (c *Catalog) Message(reason rejection.Reason) string {
	text, err := c.catalog.Resolve(reason.Code(), reason.Params())
	if err != nil {
		return reason.Code()
	}
	return text
}
