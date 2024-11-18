package models

import (
	"time"
)

type User struct {
    ID            int
    Username      string
    Email         string
    JoinDate      time.Time
    PostCount     int 
    CommentCount  int 
}