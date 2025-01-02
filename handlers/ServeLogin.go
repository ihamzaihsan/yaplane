package handlers

import (
	database "forum/database"
	"net/http"
)

func ServeLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		authPage(w, r, "login", 200, map[string]interface{}{"ErrorMessage": r.URL.Query().Get("error")})
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", 400)
		return
	}
	email, password := r.FormValue("email"), r.FormValue("password")
	if email == "" || password == "" || !database.ValidateUser(email, password) {
		oauthFailure(w, r, "Invalid email or password")
		return
	}
	if err := startSession(w, email); err != nil {
		http.Error(w, "Unable to create a session", 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
