package cookies

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

func SetLoginCookie(w http.ResponseWriter, username string) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "some_unique_value",
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

func GenerateRandomToken() string {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(token)
}
