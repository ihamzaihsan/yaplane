package handlers

import (
	"fmt"
	handleError "forum/Error"
	database "forum/database"
	"net/http"
	"strconv"
)

func AddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleError.ServeError(w, r, http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userEmail, _ := database.GetEmailFromSession(cookie.Value)
	var userID int
	err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	content := r.FormValue("content")
	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}

	if content == "" {
		handleError.ServeError(w, r, http.StatusBadRequest)
		return
	}

	result, err := database.DBInstance.DB.Exec(
		`INSERT INTO comments(content,user_id,post_id,status)
		 SELECT ?,u.id,p.id,CASE WHEN ? AND u.role='user' THEN 'pending' ELSE 'approved' END
		 FROM users u JOIN posts p ON p.id=? AND p.status='approved' WHERE u.id=?`,
		content, database.SubmissionStatus("user") == "pending", postID, userID,
	)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		moderationError(w, err)
		return
	}
	if n == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}
