package models

type ProfileData struct {
	Username     string
	Role         string
	Email        string
	JoinDate     string
	PostCount    int
	CommentCount int
	Unread       int
	IsLoggedIn   bool
}
