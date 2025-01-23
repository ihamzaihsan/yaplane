package handlers

import (
	"encoding/json"
	handleError "forum/Error"
	database "forum/database"
	"log"
	"net/http"
	"strconv"
)

func LikeDislike(w http.ResponseWriter, r *http.Request) {
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

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}

	isLike, err := strconv.ParseBool(r.FormValue("is_like"))
	if err != nil {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}

	if !publishedTarget(w, r, postID, false) {
		return
	}

	var existingIsLike bool
	err = database.DBInstance.DB.QueryRow(
		"SELECT is_like FROM likes WHERE user_id = ? AND post_id = ?",
		userID, postID,
	).Scan(&existingIsLike)

	if err == nil {

		_, err = database.DBInstance.DB.Exec(
			"UPDATE likes SET is_like = ? WHERE user_id = ? AND post_id = ?",
			isLike, userID, postID,
		)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			log.Printf("Error updating like/dislike in database for user_id: %d, post_id: %d: %v", userID, postID, err)
			return
		}
	} else {

		_, err = database.DBInstance.DB.Exec(
			`INSERT INTO likes (user_id, post_id, is_like) VALUES (?, ?, ?)`,
			userID, postID, isLike,
		)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			log.Printf("Error inserting like/dislike in database for user_id: %d, post_id: %d: %v", userID, postID, err)
			return
		}
	}

	var likesCount, dislikesCount int
	err = database.DBInstance.DB.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = true", postID,
	).Scan(&likesCount)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	err = database.DBInstance.DB.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = false", postID,
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
