package cookies

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"
)

const SessionDuration = 24 * time.Hour

func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "session_token", Value: token, Path: "/",
		Expires: time.Now().Add(SessionDuration), MaxAge: int(SessionDuration.Seconds()),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
}

func SetLoginCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, sessionCookie(token))
}

func ClearSessionCookie(w http.ResponseWriter) {
	cookie := sessionCookie("")
	cookie.Expires = time.Unix(1, 0)
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}

func GenerateRandomToken() (string, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generate session identifier: %w", err)
	}
	token[6] = token[6]&0x0f | 0x40
	token[8] = token[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", token[:4], token[4:6], token[6:8], token[8:10], token[10:]), nil
}
