package handlers

import (
	"fmt"
	handleError "forum/Error"
	database "forum/database"
	"html/template"
	"net/http"
)

func ServeUserPosts(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	isLoggedIn := false
	var userEmail string
	var userID string

	if err == nil {
		userEmail, err = database.GetEmailFromSession(cookie.Value)
		if err == nil {
			isLoggedIn = true
			userID, err = database.GetUserIDByEmail(userEmail)
			if err != nil {
				fmt.Println("Error getting user ID:", err)
			}
		}
	}

	if !isLoggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther) 
		return
	}

	posts, err := database.GetUserPosts(userID)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"isLoggedIn": isLoggedIn,
		"userEmail":  userEmail,
		"Posts":      posts,
		"ViewTitle":  "My Posts", 
	}

	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
	}
}
