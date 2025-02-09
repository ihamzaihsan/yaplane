package models

import "time"

type ActivityComment struct {
	ID, PostID             int
	Content, Title, Status string
	CanView                bool
}
type ActivityReaction struct {
	PostID, CommentID int
	Title, Content    string
	IsLike            bool
}
type ActivityPage struct {
	User      *User
	Posts     []Post
	Comments  []ActivityComment
	Reactions []ActivityReaction
}
type Notification struct {
	ID, PostID         int
	Actor, Title, Kind string
	Read               bool
	CreatedAt          time.Time
}
type NotificationPage struct {
	Items          []Notification
	Unread, Latest int
}
type EditContent struct {
	ID                                      int
	Kind, Title, Content, ImagePath, Status string
	Error                                   string
	ReviewRequired                          bool
	Topics                                  []Category
	Selected                                map[int]bool
}
