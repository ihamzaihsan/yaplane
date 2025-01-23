package handlers

import (
	database "forum/database"
	"net/http"
	"strconv"
)

func ServeDeletePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		moderationError(w, database.ErrInvalid)
		return
	}
	user, err := sessionUser(r)
	if err != nil {
		moderationError(w, err)
		return
	}
	id, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || id <= 0 {
		moderationError(w, database.ErrInvalid)
		return
	}
	if err := database.Moderate(user.ID, id, "delete-post", "", ""); err != nil {
		moderationError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
