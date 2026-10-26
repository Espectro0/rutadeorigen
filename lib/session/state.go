package session

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"
)

const (
	stateCookieName  = "ro_oauth_state"
	stateCookiePath  = "/auth"
	stateCookieTTL   = 10 * time.Minute
	stateRandomBytes = 32
)

type StateConfig struct {
	Domain string
	Secure bool
}

type State struct {
	cfg StateConfig
}

func NewState(cfg StateConfig) State {
	return State{cfg: cfg}
}

func (s State) Issue(w http.ResponseWriter) (string, error) {
	buf := make([]byte, stateRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	http.SetCookie(w, s.cookie(value, int(stateCookieTTL.Seconds())))

	return value, nil
}

func (s State) Verify(w http.ResponseWriter, r *http.Request, received string) error {
	http.SetCookie(w, s.cookie("", -1))

	cookie, err := r.Cookie(stateCookieName)
	if err != nil || cookie.Value == "" || received == "" {
		return ErrInvalidState
	}
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(received)) != 1 {
		return ErrInvalidState
	}

	return nil
}

func (s State) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     stateCookieName,
		Value:    value,
		Path:     stateCookiePath,
		Domain:   s.cfg.Domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}
