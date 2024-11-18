package cookies

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func GetSessionCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, fmt.Errorf("cookie not found: %v", err)
	}
	return cookie, nil
}
func GenerateRandomToken() string {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		panic(err) 
	}
	return hex.EncodeToString(token)
}
