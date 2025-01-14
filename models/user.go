package models

import (
	"time"
)

type User struct {
	ID           int
	Username     string
	Role         string
	Email        string
	JoinDate     time.Time
	PostCount    int
	CommentCount int
}
