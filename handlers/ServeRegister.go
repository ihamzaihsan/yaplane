package handlers

import (
	handleError "forum/Error"
	cookies "forum/cookies"
	database "forum/database"
	"html/template"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func ServeRegister(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		tmpl, err := template.ParseFiles("static/register.html")
		if err != nil {
			handleError.ServeError(w, r, 500)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			handleError.ServeError(w, r, 500)
		}
		return
	}

	if r.Method == http.MethodPost {

		err := r.ParseForm()
		if err != nil {
			handleError.ServeError(w, r, 400)
		}

		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := strings.TrimSpace(r.FormValue("password"))
		confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))

		tmpl, err := template.ParseFiles("static/register.html")
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		if username == "" || email == "" || password == "" || confirmPassword == "" {
			errMsg := "All fields are required or can't be empty spaces."
			tmpl.Execute(w, map[string]interface{}{
				"Error": errMsg,
			})
			return
		}

		
		if password != confirmPassword {
			errMsg := "Passwords do not match."
			tmpl.Execute(w, map[string]interface{}{
				"Error": errMsg,
			})
			return
		}

		
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error securing password", http.StatusInternalServerError)
			return
		}

	
		_, err = database.DBInstance.DB.Exec(
			"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			username, email, hashedPassword,
		)
		if err != nil {
			
			if err.Error() == "UNIQUE constraint failed: users.username" || err.Error() == "UNIQUE constraint failed: users.email" {
				errMsg := "Username or email already exists."
				tmpl.Execute(w, map[string]interface{}{
					"Error": errMsg,
				})
			} else {
				handleError.ServeError(w, r, http.StatusInternalServerError)
			}
			return
		}

		sessionToken := cookies.GenerateRandomToken()

		err = database.StoreSession(sessionToken, email)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}
