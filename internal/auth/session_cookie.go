package auth

import (
	"net/http"
	"strings"
	"time"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

const SessionCookieName = "arx_session"

// SessionCookieConfig controls JWT session cookie attributes.
type SessionCookieConfig struct {
	SameSite string
	Secure   *bool
	MaxAge   int
}

// SessionCookieConfigFromSecurity builds cookie policy from server security settings.
func SessionCookieConfigFromSecurity(sec arxconfig.SecurityConfig) SessionCookieConfig {
	maxAge := sec.TokenExpirationHours * 3600
	if maxAge <= 0 {
		maxAge = 24 * 3600
	}
	sameSite := strings.TrimSpace(sec.CookieSameSite)
	if sameSite == "" {
		sameSite = "lax"
	}
	return SessionCookieConfig{
		SameSite: sameSite,
		Secure:   sec.CookieSecure,
		MaxAge:   maxAge,
	}
}

func (c SessionCookieConfig) resolveSameSite() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(c.SameSite)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (c SessionCookieConfig) isSecure(r *http.Request) bool {
	if c.Secure != nil {
		return *c.Secure
	}
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	return false
}

// SetSessionCookie writes the HttpOnly JWT session cookie for browser clients.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, cfg SessionCookieConfig, token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cfg.MaxAge,
		HttpOnly: true,
		Secure:   cfg.isSecure(r),
		SameSite: cfg.resolveSameSite(),
		Expires:  time.Now().Add(time.Duration(cfg.MaxAge) * time.Second),
	})
}

// ClearSessionCookie removes the JWT session cookie.
func ClearSessionCookie(w http.ResponseWriter, r *http.Request, cfg SessionCookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.isSecure(r),
		SameSite: cfg.resolveSameSite(),
		Expires:  time.Unix(0, 0),
	})
}

// SessionTokenFromRequest returns the JWT from the session cookie when present.
func SessionTokenFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie == nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", false
	}
	return token, true
}
