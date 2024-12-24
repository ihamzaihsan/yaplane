package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/mail"
)

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && len(email) <= 254
}

func authPage(w http.ResponseWriter, r *http.Request, page string, status int, data map[string]interface{}) {
	_, providers, err := oauthSettings()
	if err == nil {
		_, data["GoogleEnabled"] = providers["google"]
		_, data["GitHubEnabled"] = providers["github"]
	}
	t, err := template.ParseFiles("static/" + page + ".html")
	if err != nil {
		http.Error(w, "Unable to load account page", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		// Headers may already be sent; avoid appending a second HTML document.
		log.Printf("Render account page: %v", err)
		return
	}
}
