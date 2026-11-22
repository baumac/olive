package publicapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// sessionCookieName is the name of the cookieAuth scheme in the spec.
const sessionCookieName = "reviews_session"

var crossOrigin = http.NewCrossOriginProtection()

// sameOrigin is false for an unsafe request that a browser sent from
// another origin. Decision 00003 has the rules.
func sameOrigin(r *http.Request) bool {
	return crossOrigin.Check(r) == nil
}

// cookieToken returns the token of the session cookie. The Authorization
// header wins over the cookie. A cross-origin request has no cookie token.
func cookieToken(st *state) string {
	r := st.request
	if r.Header.Get("Authorization") != "" || !st.sameOrigin {
		return ""
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// wantsCookie reports if the response puts the token in the session cookie.
func wantsCookie(st *state, param *bool) bool {
	return st.fromCookie || (param != nil && *param && st.sameOrigin)
}

func (s *Server) sessionCookie(r *http.Request, token string, expires time.Time) string {
	return s.cookie(r, token, max(1, int(time.Until(expires).Seconds())))
}

func (s *Server) deleteSessionCookie(r *http.Request) string {
	return s.cookie(r, "", -1)
}

func (s *Server) cookie(r *http.Request, value string, maxAge int) string {
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     basePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure(r),
	}
	return c.String()
}

// secure reports if the browser sent r over https. Behind a reverse proxy,
// the request has no TLS. X-Forwarded-Proto or an https Application URL with
// the host of r then shows the scheme.
func (s *Server) secure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// A client can only send a wrong value for its own cookie.
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.EqualFold(proto, "https")
	}
	u, err := url.Parse(s.app.Settings().Meta.AppURL)
	return err == nil && u.Scheme == "https" && strings.EqualFold(u.Host, r.Host)
}
