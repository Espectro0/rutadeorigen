package middleware

import (
	"net/http"
	"rutadeorigen/lib/requestcontext"

	"github.com/google/uuid"
)

const (
	CorrelationIDHeader  = "X-Correlation-ID"
	maxCorrelationIDSize = 64
)

func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(CorrelationIDHeader)
		if !isValidCorrelationID(id) {
			id = uuid.NewString()
		}

		w.Header().Set(CorrelationIDHeader, id)
		ctx := requestcontext.WithCorrelationID(r.Context(), id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isValidCorrelationID(id string) bool {
	if id == "" || len(id) > maxCorrelationIDSize {
		return false
	}
	for _, c := range id {
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isLetter && !isDigit && c != '-' && c != '_' {
			return false
		}
	}
	return true
}
