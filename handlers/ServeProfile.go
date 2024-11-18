package handlers

import (
	database "forum/database"
	ProfileData "forum/models"
	"html/template"
	"log"
	"net/http"
)

func ServeProfile(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_token")
	if err != nil {
	
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionToken := cookie.Value
	user, err := database.GetUserBySession(sessionToken) 
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	
	profileData := ProfileData.ProfileData{
		Username:     user.Username,
		Email:        user.Email,
		JoinDate:     user.JoinDate.Format("January 2, 2006"),
		PostCount:    user.PostCount,
		CommentCount: user.CommentCount,
		IsLoggedIn:   true, 
	}

	tmpl, err := template.ParseFiles("static/templates/profile.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, profileData)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
