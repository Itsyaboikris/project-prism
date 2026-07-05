package auth

import (
	"net/http"
	"strings"
	"time"
)

func (s *Service) RefreshTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(s.config.RefreshCookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func (s *Service) SetRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.RefreshCookieName,
		Value:    token,
		Path:     s.config.RefreshCookiePath,
		Domain:   s.config.RefreshCookieDomain,
		HttpOnly: true,
		Secure:   s.config.RefreshCookieSecure,
		SameSite: sameSiteFromConfig(s.config.RefreshCookieSameSite),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (s *Service) ClearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.RefreshCookieName,
		Value:    "",
		Path:     s.config.RefreshCookiePath,
		Domain:   s.config.RefreshCookieDomain,
		HttpOnly: true,
		Secure:   s.config.RefreshCookieSecure,
		SameSite: sameSiteFromConfig(s.config.RefreshCookieSameSite),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func sameSiteFromConfig(v string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
