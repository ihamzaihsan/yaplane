package handlers

import (
	handleError "forum/Error"
	database "forum/database"
	"forum/models"
	"html/template"
	"log"
	"net/http"
	"strings"
)


func ServeMainForum(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		handleError.ServeError(w, r, http.StatusNotFound)
		return
	}

	if !database.ValidateCategoriesPath(r) {
		handleError.ServeError(w, r, http.StatusNotFound)
		return
	}


	categories := r.URL.Query()["categories"]

	w.Header().Set("Cache-Control", "no-cache, must-revalidate")

	cookie, err := r.Cookie("session_token")
	isLoggedIn := false
	var userEmail string

	if err == nil {
		userEmail, err = database.GetEmailFromSession(cookie.Value)
		if err == nil {
			isLoggedIn = true
		}
	}

	var posts []models.Post
	var viewTitle string

	if len(categories) == 0 {
		posts, err = database.GetLatestPosts()
		viewTitle = "Latest Posts"
	} else {
		posts, err = database.GetPostsByCategories(categories)
		viewTitle = "Posts in " + strings.Join(categories, ", ")
	}

	if err != nil {
		handleError.ServeError(w, r, http.StatusInternalServerError)
		log.Println("error: ", err)
		return
	}

	data := map[string]interface{}{
		"isLoggedIn": isLoggedIn,
		"userEmail":  userEmail,
		"Posts":      posts,
		"ViewTitle":  viewTitle,
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
