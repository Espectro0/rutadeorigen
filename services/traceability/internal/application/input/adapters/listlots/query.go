package listlots

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

func toQuery(req request.ListLots) (listlotsuc.Query, error) {
	var q listlotsuc.Query

	if v := helper.ApplyTrim(req.FarmID); v != "" {
		id := helper.ParseUUID(v)
		if helper.IsDefaultUUID(id) {
			return q, exception.New(exception.CodeLotInvalidFarmID)
		}
		q.Filter.FarmID = id
	}

	if v := helper.ApplyTrim(req.CustodianID); v != "" {
		id := helper.ParseUUID(v)
		if helper.IsDefaultUUID(id) {
			return q, exception.New(exception.CodeLotInvalidCustodianID)
		}
		q.Filter.CustodianID = id
	}

	if v := helper.ApplyTrim(req.Limit); v != "" {
		n := helper.ParseInt(v)
		if n < 1 {
			return q, exception.New(exception.CodeInvalidLimit)
		}
		q.Limit = n
	}

	if v := helper.ApplyTrim(req.Cursor); v != "" {
		c, err := decodeCursor(v)
		if err != nil {
			return q, err
		}
		q.After = c
	}
	return q, nil
}

func encodeCursor(c lot.Cursor) string {
	if c.IsDefault() {
		return ""
	}
	raw := strconv.FormatInt(c.CreatedAt.UnixNano(), 10) + "." + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(value string) (lot.Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return lot.Cursor{}, exception.New(exception.CodeInvalidCursor)
	}

	nanos, idPart, ok := strings.Cut(string(raw), ".")
	n, parseErr := strconv.ParseInt(nanos, 10, 64)
	id := helper.ParseUUID(idPart)
	if !ok || parseErr != nil || helper.IsDefaultUUID(id) {
		return lot.Cursor{}, exception.New(exception.CodeInvalidCursor)
	}
	return lot.Cursor{CreatedAt: time.Unix(0, n).UTC(), ID: id}, nil
}
