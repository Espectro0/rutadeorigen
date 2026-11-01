package ifmatch

import (
	"strconv"
	"strings"

	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

func Parse(header string) (int, error) {
	value := strings.TrimPrefix(helper.ApplyTrim(header), "W/")
	value = strings.Trim(value, `"`)

	version, err := strconv.Atoi(value)
	if err != nil || version < 1 {
		return 0, exception.New(exception.CodeLotInvalidIfMatch)
	}
	return version, nil
}
