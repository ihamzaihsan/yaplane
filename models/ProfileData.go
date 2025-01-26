package models

type ProfileData struct {
	Username     string
	Role         string
	Email        string
	JoinDate     string
	PostCount    int
	CommentCount int
	IsLoggedIn   bool
}
