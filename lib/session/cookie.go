package session

import (
	"net/http"
	"time"
)

const (
	defaultCookieName   = "ro_session"
	defaultCookiePath   = "/"
	defaultCookieMaxAge = 30 * 24 * time.Hour
	maxCookieValueBytes = 4096
)

type CookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
	MaxAge   time.Duration
}

type Cookie struct {
	cfg CookieConfig
}

func NewCookie(cfg CookieConfig) Cookie {
	if cfg.Name == "" {
		cfg.Name = defaultCookieName
	}
	if cfg.Path == "" {
		cfg.Path = defaultCookiePath
	}
	if cfg.SameSite == 0 {
		cfg.SameSite = http.SameSiteLaxMode
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = defaultCookieMaxAge
	}
	return Cookie{cfg: cfg}
}

func (c Cookie) Read(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(c.cfg.Name)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func (c Cookie) Set(w http.ResponseWriter, sealed string) error {
	if len(sealed) > maxCookieValueBytes {
		return ErrCookieTooLarge
	}
	http.SetCookie(w, &http.Cookie{
		Name:     c.cfg.Name,
		Value:    sealed,
		Path:     c.cfg.Path,
		Domain:   c.cfg.Domain,
		MaxAge:   int(c.cfg.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   c.cfg.Secure,
		SameSite: c.cfg.SameSite,
	})
	return nil
}

func (c Cookie) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.cfg.Name,
		Value:    "",
		Path:     c.cfg.Path,
		Domain:   c.cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.cfg.Secure,
		SameSite: c.cfg.SameSite,
	})
}
