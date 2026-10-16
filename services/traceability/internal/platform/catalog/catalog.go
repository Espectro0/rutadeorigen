package catalog

import (
	_ "embed"
	"fmt"

	libcatalog "rutadeorigen/lib/catalog"
)

//go:embed messages.yaml
var messagesYAML []byte

func New() (*libcatalog.Catalog, error) {
	c, err := libcatalog.NewWithGeneric(messagesYAML)
	if err != nil {
		return nil, fmt.Errorf("traceability catalog: %w", err)
	}

	return c, nil
}
