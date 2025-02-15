package handlers

import (
	"database/sql"
	"errors"
	database "forum/database"
	"forum/models"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/mattn/go-sqlite3"
)

func sessionUser(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, err
	}
	return database.GetUserBySession(cookie.Value)
}

func moderationError(w http.ResponseWriter, err error) {
	status := 500
	var sqliteErr sqlite3.Error
	switch {
	case errors.Is(err, database.ErrForbidden):
		status = 403
	case errors.Is(err, database.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		status = 404
	case errors.Is(err, database.ErrInvalid):
		status = 400
	case errors.Is(err, database.ErrConflict):
		status = 409
	case errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint:
		status = 409
	}
	message := err.Error()
	if status == 500 {
		log.Printf("Moderation: %v", err)
		message = "Unable to complete this action."
	}
	if status == 409 {
		message = "This item already exists, has been handled, or is still in use. Go back and refresh the page."
	}
	http.Error(w, message, status)
}

func ServeModeration(w http.ResponseWriter, r *http.Request) {
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	data, err := database.ModerationData(user)
	if err != nil {
		moderationError(w, err)
		return
	}
	t, err := template.ParseFiles("static/moderation.html")
	if err != nil {
		moderationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		log.Printf("Moderation page: %v", err)
	}
}

func ServeModerationAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		moderationError(w, database.ErrInvalid)
		return
	}
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	action := r.FormValue("action")
	id := 0
	if action != "request-role" && action != "create-category" {
		id, err = strconv.Atoi(r.FormValue("id"))
		if err != nil || id <= 0 {
			moderationError(w, database.ErrInvalid)
			return
		}
	}
	if err := database.Moderate(user.ID, id, action, r.FormValue("text"), r.FormValue("reason")); err != nil {
		moderationError(w, err)
		return
	}
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func postFormData(data map[string]interface{}) (map[string]interface{}, error) {
	categories, err := database.GetAllCategories()
	data["TopicOptions"] = categories
	data["Premoderate"] = database.SubmissionStatus("user") == "pending"
	return data, err
}

func executePostForm(w http.ResponseWriter, r *http.Request, t *template.Template, data map[string]interface{}) {
	data, err := postFormData(data)
	if err != nil {
		moderationError(w, err)
		return
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("Post form: %v", err)
	}
}

func addTopics(data map[string]interface{}) error {
	categories, err := database.GetAllCategories()
	data["TopicOptions"] = categories
	return err
}
