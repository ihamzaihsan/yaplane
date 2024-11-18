package handlers

import (
	handleError "forum/Error"
	database "forum/database"
	"forum/models"
	"html/template"
	"net/http"
	"strings"
)

func ServeIndividualPost(w http.ResponseWriter, r *http.Request) {

	var currentUser string
	cookie, err := r.Cookie("session_token")
	if err == nil {
		userEmail, _ := database.GetEmailFromSession(cookie.Value)
		database.DBInstance.DB.QueryRow("SELECT username FROM users WHERE email = ?", userEmail).Scan(&currentUser)
	}

	postID := strings.TrimPrefix(r.URL.Path, "/post/")

	
	post, comments, likes, dislikes := database.GetPostDetails(postID)


	if post.ID == 0 {
		handleError.ServeError(w, r, http.StatusNotFound)
		return
	}

	data := struct {
		Post        models.Post
		Comments    []models.Comment
		Likes       int
		Dislikes    int
		CurrentUser string
	}{
		Post:        post,
		Comments:    comments,
		Likes:       likes,
		Dislikes:    dislikes,
		CurrentUser: currentUser,
	}


	tmpl, err := template.ParseFiles("static/templates/post.html")
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
	}
}
