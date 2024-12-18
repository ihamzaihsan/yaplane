package Forum

import (
	"database/sql"
	"errors"
	"forum/cookies"
	"strings"
)

var ErrEmailRegistered = errors.New("this email already has an account; use its original sign-in method")

// OAuthUser binds a stable provider identity to an account, never just its email.
func OAuthUser(provider, subject, email, name string) (string, error) {
	tx, err := DBInstance.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var existing string
	err = tx.QueryRow(`SELECT u.email FROM oauth_identities i JOIN users u ON u.id=i.user_id
		WHERE i.provider=? AND i.subject=?`, provider, subject).Scan(&existing)
	if err == nil {
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE email=? COLLATE NOCASE", email).Scan(&count); err != nil {
		return "", err
	}
	if count != 0 {
		return "", ErrEmailRegistered
	}
	token, err := cookies.GenerateRandomToken()
	if err != nil {
		return "", err
	}
	// An empty hash deliberately disables password login for provider-only accounts.
	label := []rune(strings.Join(strings.Fields(name), "_"))
	if len(label) > 32 {
		label = label[:32]
	}
	username := string(label) + "_" + token[:8]
	if len(label) == 0 {
		username = provider + "_" + token[:8]
	}
	result, err := tx.Exec("INSERT INTO users(username,email,password) VALUES(?,?,'')", username, email)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec("INSERT INTO oauth_identities(provider,subject,user_id) VALUES(?,?,?)", provider, subject, id); err != nil {
		return "", err
	}
	return email, tx.Commit()
}
