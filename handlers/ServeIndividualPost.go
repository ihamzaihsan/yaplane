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
	var viewer *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		viewer, _ = database.GetUserBySession(cookie.Value)
		if viewer != nil {
			currentUser = viewer.Username
		}
	}

	postID := strings.TrimPrefix(r.URL.Path, "/post/")

	if err := database.CanViewPost(postID, viewer); err != nil {
		moderationError(w, err)
		return
	}
	post, comments, likes, dislikes := database.GetPostDetails(postID, viewer)

	if post.ID == 0 {
		handleError.ServeError(w, r, http.StatusNotFound)
		return
	}

	data := struct {
		Post                     models.Post
		Comments                 []models.Comment
		Likes                    int
		Dislikes                 int
		CurrentUser              string
		Staff, Admin, IsLoggedIn bool
		Premoderate              bool
	}{
		Post:        post,
		Comments:    comments,
		Likes:       likes,
		Dislikes:    dislikes,
		CurrentUser: currentUser,
		IsLoggedIn:  viewer != nil,
	}
	if viewer != nil {
		data.Staff = viewer.Role == "moderator" || viewer.Role == "admin"
		data.Admin = viewer.Role == "admin"
		data.Premoderate = database.SubmissionStatus(viewer.Role) == "pending"
	}

	tmpl, err := template.ParseFiles("static/post.html")
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
	}
}
