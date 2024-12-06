package middleware

import (
	"database/sql"
	"errors"
	"forum/cookies"
	database "forum/database"
	"net/http"
	"net/url"
	"strings"
)

func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err == nil {
			_, err = database.GetEmailFromSession(cookie.Value)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Unable to validate the session", http.StatusInternalServerError)
				return
			}
		}
		if err != nil {
			cookies.ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Security protects existing HTML forms and fetch requests without new frontend code.
// Mutations must originate from the forum; missing or opaque origins are rejected.
func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "base-uri 'self'; frame-ancestors 'none'; object-src 'none'")
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			u, err := url.Parse(origin)
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			if err != nil || u.User != nil || u.Scheme != scheme || !strings.EqualFold(u.Host, r.Host) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "This request must come from the forum.", http.StatusForbidden)
				return
			}
			if r.URL.Path != "/posts/create" {
				r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			}
		}
		next.ServeHTTP(w, r)
	})
}
