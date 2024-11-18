package handlers

import (
	cookies "forum/cookies"
	database "forum/database"
	"html/template"
	"log"
	"net/http"
	"time"
)

func ServeLogin(w http.ResponseWriter, r *http.Request) {
	
	if r.Method == http.MethodGet {
		
		tmpl, err := template.ParseFiles("static/login.html")
		if err != nil {
			http.Error(w, "Error loading login page", http.StatusInternalServerError)
			return
		}

		
		errorMessage := r.URL.Query().Get("error")

		
		data := map[string]interface{}{
			"ErrorMessage": errorMessage,
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Error rendering login page", http.StatusInternalServerError)
			return
		}
		return
	}


	if r.Method == http.MethodPost {
		
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		
		isValid := database.ValidateUser(email, password)
		if isValid {
		
			sessionToken := cookies.GenerateRandomToken()

			err := database.StoreSession(sessionToken, email)
			if err != nil {
				http.Error(w, "user already logged in or Error storing session", http.StatusInternalServerError)
				log.Println("error: ", err)
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

		
		http.Redirect(w, r, "/login?error=Invalid email or password", http.StatusSeeOther)
		return
	}

	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
