package handlers

import (
	"encoding/json"
	database "forum/database"
	"net/http"
	"strconv"
)

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
