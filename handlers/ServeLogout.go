package handlers

import (
	database "forum/database"
	"log"
	"net/http"
	"time"
)

func ServeLogout(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}


	cookie, err := r.Cookie("session_token")
	if err != nil {
	
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}


	_, err = database.DBInstance.DB.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
	if err != nil {
		log.Printf("Failed to delete session from database: %v", err)
	}


	expiredCookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), 
		HttpOnly: true,
	}
	http.SetCookie(w, expiredCookie)


	http.Redirect(w, r, "/", http.StatusSeeOther)
}
