package handlers

import (
	"encoding/json"
	database "forum/database"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

func activityPage(w http.ResponseWriter, name string, status int, data interface{}) {
	t, err := template.ParseFiles("static/" + name + ".html")
	if err != nil {
		moderationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		log.Printf("Render %s: %v", name, err)
	}
}

func ServeActivity(w http.ResponseWriter, r *http.Request) {
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	data, err := database.Activity(user)
	if err != nil {
		moderationError(w, err)
		return
	}
	activityPage(w, "activity", 200, data)
}

func ServeNotifications(w http.ResponseWriter, r *http.Request) {
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	data, err := database.Notifications(user.ID)
	if err != nil {
		moderationError(w, err)
		return
	}
	activityPage(w, "notifications", 200, data)
}

func ServeNotificationCount(w http.ResponseWriter, r *http.Request) {
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	unread, latest, err := database.NotificationCounts(user.ID)
	if err != nil {
		moderationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unread": unread, "latest": latest})
}

func ServeReadNotifications(w http.ResponseWriter, r *http.Request) {
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	if err := database.ReadNotifications(user.ID); err != nil {
		moderationError(w, err)
		return
	}
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

func reaction(w http.ResponseWriter, r *http.Request, comment bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if err := r.ParseForm(); err != nil {
		moderationError(w, database.ErrInvalid)
		return
	}
	field := "post_id"
	if comment {
		field = "comment_id"
	}
	id, err := strconv.Atoi(r.FormValue(field))
	if err != nil || id <= 0 {
		moderationError(w, database.ErrInvalid)
		return
	}
	isLike, err := strconv.ParseBool(r.FormValue("is_like"))
	if err != nil {
		moderationError(w, database.ErrInvalid)
		return
	}
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	likes, dislikes, err := database.RecordReaction(user.ID, id, comment, isLike)
	if err != nil {
		moderationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Likes    int `json:"likes"`
		Dislikes int `json:"dislikes"`
	}{likes, dislikes})
}
