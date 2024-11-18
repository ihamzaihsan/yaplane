package handlers

import (
	"encoding/json" 
	handleError "forum/Error"
	database "forum/database"
	"log"
	"net/http"
	"strconv"
)

func LikeDislikeComment(w http.ResponseWriter, r *http.Request) {
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
		handleError.ServeError(w, r, http.StatusInternalServerError)
		log.Println("Error getting user email from session:", err)
		return
	}

	var userID int
	err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		log.Println("Error getting user ID from database:", err)
		return
	}


	commentID, err := strconv.Atoi(r.FormValue("comment_id"))
	if err != nil && r.FormValue("comment_id") != "" {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}

	isLike, err := strconv.ParseBool(r.FormValue("is_like"))
	if err != nil {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}


	if commentID == 0 {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}


	var existingIsLike bool
	err = database.DBInstance.DB.QueryRow(
		"SELECT is_like FROM likes WHERE user_id = ? AND comment_id = ?",
		userID, commentID,
	).Scan(&existingIsLike)

	if err == nil {
		
		_, err = database.DBInstance.DB.Exec(
			"UPDATE likes SET is_like = ? WHERE user_id = ? AND comment_id = ?",
			isLike, userID, commentID,
		)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			log.Printf("Error updating like/dislike in database for user_id: %d, comment_id: %d: %v", userID, commentID, err)
			return
		}
	} else {
		
		_, err = database.DBInstance.DB.Exec(
			`INSERT INTO likes (user_id, comment_id, is_like)
             VALUES (?, ?, ?)`,
			userID, commentID, isLike,
		)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			log.Printf("Error inserting like/dislike in database for user_id: %d, comment_id: %d: %v", userID, commentID, err)
			return
		}
	}


	var likesCount, dislikesCount int
	err = database.DBInstance.DB.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = true", commentID,
	).Scan(&likesCount)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	err = database.DBInstance.DB.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = false", commentID,
	).Scan(&dislikesCount)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	response := struct {
		Likes    int `json:"likes"`
		Dislikes int `json:"dislikes"`
	}{
		Likes:    likesCount,
		Dislikes: dislikesCount,
	}


	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
