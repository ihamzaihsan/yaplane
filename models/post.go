package models

import "time"

type Post struct {
	ID         int
	Title      string
	Content    string
	Username   string
	CreatedAt  time.Time
	Likes      int
	Dislikes   int
	Categories []string
	ImagePath  string
	UserReaction     string 
}
