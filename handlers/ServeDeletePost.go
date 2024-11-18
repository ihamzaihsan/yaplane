package handlers

import (
	handleError "forum/Error"
	database "forum/database"
	"net/http"
)

func ServeDeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleError.ServeError(w, r, http.StatusMethodNotAllowed)
		return
	}

	
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	
	userEmail, err := database.GetEmailFromSession(cookie.Value)
	if err != nil {
		handleError.ServeError(w, r, http.StatusUnauthorized)
		return
	}


	postID := r.FormValue("post_id")
	if postID == "" {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}


	var postOwnerEmail string
	err = database.DBInstance.DB.QueryRow(
		`SELECT u.email 
         FROM posts p 
         JOIN users u ON p.user_id = u.id 
         WHERE p.id = ?`, postID).Scan(&postOwnerEmail)

	if err != nil {
		handleError.ServeError(w, r, http.StatusNotFound)
		return
	}

	
	if postOwnerEmail != userEmail {
		handleError.ServeError(w, r, http.StatusForbidden)
		return
	}

	_, err = database.DBInstance.DB.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}


	http.Redirect(w, r, "/", http.StatusSeeOther)
}
