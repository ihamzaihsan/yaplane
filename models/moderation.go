package models

type RoleRequest struct {
	UserID                  int
	Username, Status, Reply string
}
type ModerationReport struct {
	ID, PostID                              int
	Reporter, Title, Reason, Message, Reply string
}
type PendingContent struct {
	ID, PostID                     int
	Kind, Title, Content, Username string
}
type ModerationPage struct {
	User         *User
	Staff, Admin bool
	Requests     []RoleRequest
	Reports      []ModerationReport
	Pending      []PendingContent
	Users        []User
	Categories   []Category
}
