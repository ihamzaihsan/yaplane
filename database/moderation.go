package Forum

import (
	"database/sql"
	"errors"
	"fmt"
	"forum/models"
	"os"
	"strings"
)

var (
	ErrForbidden = errors.New("you do not have permission for this action")
	ErrNotFound  = errors.New("the requested item was not found")
	ErrInvalid   = errors.New("check the submitted fields")
	ErrConflict  = errors.New("this item has already been handled or is still in use")
)

// Add only missing columns, keeping existing accounts and content intact.
func migrateModeration(db *sql.DB) error {
	for _, c := range []struct{ table, name, definition string }{
		{"users", "role", "TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user','moderator','admin'))"},
		{"posts", "status", "TEXT NOT NULL DEFAULT 'approved' CHECK(status IN ('approved','pending'))"},
		{"comments", "status", "TEXT NOT NULL DEFAULT 'approved' CHECK(status IN ('approved','pending'))"},
	} {
		rows, err := db.Query("PRAGMA table_info(" + c.table + ")")
		if err != nil {
			return err
		}
		found := false
		for rows.Next() {
			var id, required, primary int
			var name, kind string
			var fallback interface{}
			if err := rows.Scan(&id, &name, &kind, &required, &fallback, &primary); err != nil {
				rows.Close()
				return err
			}
			found = found || name == c.name
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if !found {
			if _, err := db.Exec("ALTER TABLE " + c.table + " ADD COLUMN " + c.name + " " + c.definition); err != nil {
				return err
			}
		}
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS moderator_requests (
		user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','declined')),
		reply TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS moderation_reports (
		id INTEGER PRIMARY KEY, post_id INTEGER REFERENCES posts(id) ON DELETE SET NULL,
		reporter_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL, reason TEXT NOT NULL CHECK(reason IN ('irrelevant','obscene','illegal','insulting')),
		message TEXT NOT NULL, reply TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS open_moderation_report ON moderation_reports(post_id,reporter_id) WHERE reply='';
	CREATE INDEX IF NOT EXISTS pending_posts ON posts(status);
	CREATE INDEX IF NOT EXISTS pending_comments ON comments(status);`)
	return err
}

func BootstrapAdmin(db *sql.DB, email string) error {
	result, err := db.Exec("UPDATE users SET role='admin' WHERE email=?", email)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func CreateUser(username, email string, hash []byte, requestModerator bool) error {
	tx, err := DBInstance.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec("INSERT INTO users(username,email,password) VALUES(?,?,?)", username, email, hash)
	if err != nil {
		return err
	}
	if requestModerator {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO moderator_requests(user_id) VALUES(?)", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func SubmissionStatus(role string) string {
	if strings.EqualFold(os.Getenv("FORUM_PREMODERATE"), "true") && role == "user" {
		return "pending"
	}
	return "approved"
}

// Recheck the role inside each write transaction so demotion takes effect immediately.
func CanViewPost(id string, user *models.User) error {
	actor, staff := 0, false
	if user != nil {
		actor = user.ID
		staff = user.Role == "admin" || user.Role == "moderator"
	}
	var exists bool
	err := DBInstance.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id=? AND (status='approved' OR user_id=? OR ?))", id, actor, staff).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func CanViewUpload(path string, user *models.User) error {
	actor, staff := 0, false
	if user != nil {
		actor = user.ID
		staff = user.Role == "admin" || user.Role == "moderator"
	}
	var hidden bool
	err := DBInstance.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE image_path=? AND status='pending' AND user_id!=? AND NOT ?)", path, actor, staff).Scan(&hidden)
	if err != nil {
		return fmt.Errorf("check upload visibility: %w", err)
	}
	if hidden {
		return ErrNotFound
	}
	return nil
}
