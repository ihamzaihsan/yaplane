package handlers

import (
	"encoding/json"
	"errors"
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

func targetID(r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return 0, database.ErrInvalid
	}
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id <= 0 {
		return 0, database.ErrInvalid
	}
	return id, nil
}

func ServeEditPost(w http.ResponseWriter, r *http.Request)    { editContent(w, r, "post") }
func ServeEditComment(w http.ResponseWriter, r *http.Request) { editContent(w, r, "comment") }

func editContent(w http.ResponseWriter, r *http.Request, kind string) {
	id, err := targetID(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	item, err := database.Editable(user.ID, id, kind)
	if err != nil {
		moderationError(w, err)
		return
	}
	item.ReviewRequired = database.SubmissionStatus(user.Role) == "pending"
	if r.Method == http.MethodPost {
		item.Title, item.Content = r.FormValue("title"), r.FormValue("content")
		categories := []int{}
		item.Selected = make(map[int]bool)
		for _, value := range r.PostForm["category_ids[]"] {
			category, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				err = database.ErrInvalid
				break
			}
			categories = append(categories, category)
			item.Selected[category] = true
		}
		if err == nil {
			err = database.EditOwned(user.ID, id, kind, item.Title, item.Content, categories)
		}
		if errors.Is(err, database.ErrInvalid) {
			item.Error = "Enter valid text and topics. Titles are limited to 200 bytes and content to 20,000 bytes."
			activityPage(w, "edit", 400, item)
			return
		}
		if err != nil {
			moderationError(w, err)
			return
		}
		http.Redirect(w, r, "/activity", http.StatusSeeOther)
		return
	}
	activityPage(w, "edit", 200, item)
}

func ServeDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, err := targetID(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	if err := database.DeleteOwnComment(user.ID, id); err != nil {
		moderationError(w, err)
		return
	}
	http.Redirect(w, r, "/activity", http.StatusSeeOther)
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
