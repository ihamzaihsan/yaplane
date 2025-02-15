package handlers

import "net/http"

func LikeDislikeComment(w http.ResponseWriter, r *http.Request) { reaction(w, r, true) }
