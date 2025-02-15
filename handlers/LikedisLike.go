package handlers

import "net/http"

func LikeDislike(w http.ResponseWriter, r *http.Request) { reaction(w, r, false) }
