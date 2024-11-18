package Forum

import (
	"database/sql"
	"errors"
	"fmt"
	models "forum/models"
	"time"
)

func StoreSession(sessionToken, email string) error {
	tx, err := DBInstance.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM sessions WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("failed to delete existing session: %v", err)
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	_, err = tx.Exec(
		"INSERT INTO sessions (session_token, email, expires_at) VALUES (?, ?, ?)",
		sessionToken, email, expirationTime,
	)
	if err != nil {
		return fmt.Errorf("failed to store session: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	return nil
}

func GetEmailFromSession(token string) (string, error) {
	var email string
	err := DBInstance.DB.QueryRow("SELECT email FROM sessions WHERE session_token = ?", token).Scan(&email)
	if err != nil {
		return "", err
	}
	return email, nil
}

func GetUserBySession(sessionToken string) (*models.User, error) {
	var user models.User

	query := `SELECT u.id, u.username, u.email, u.created_at, 
                     COALESCE((SELECT COUNT(*) FROM posts WHERE user_id = u.id), 0) AS post_count,
                     COALESCE((SELECT COUNT(*) FROM comments WHERE user_id = u.id), 0) AS comment_count
              FROM users u 
              JOIN sessions s ON u.email = s.email 
              WHERE s.session_token = ?`
	row := DBInstance.DB.QueryRow(query, sessionToken)
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.JoinDate, &user.PostCount, &user.CommentCount)
	if err == sql.ErrNoRows {
		return nil, errors.New("no user found with the provided session token")
	} else if err != nil {
		return nil, err
	}
	return &user, nil
}

func CheckActiveSession(email string) (string, error) {
	var sessionToken string
	err := DBInstance.DB.QueryRow("SELECT session_token FROM sessions WHERE email = ? AND expires_at > ?", email, time.Now()).Scan(&sessionToken)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return sessionToken, nil
}
