package handlers

import (
	"forum/cookies"
	database "forum/database"
	"net/http"
)

func startSession(w http.ResponseWriter, email string) error {
	token, err := cookies.GenerateRandomToken()
	if err != nil {
		return err
	}
	if err := database.StoreSession(token, email); err != nil {
		return err
	}
	cookies.SetLoginCookie(w, token)
	return nil
}
