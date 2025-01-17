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
func Moderate(actor, id int, action, text, reason string) error {
	tx, err := DBInstance.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var role string
	if err := tx.QueryRow("SELECT role FROM users WHERE id=?", actor).Scan(&role); err != nil {
		return err
	}
	staff := role == "moderator" || role == "admin"
	admin := role == "admin"
	text = strings.TrimSpace(text)
	if len(text) > 2000 {
		return ErrInvalid
	}
	var result sql.Result
	switch action {
	case "request-role":
		if role != "user" {
			return ErrForbidden
		}
		var status string
		err := tx.QueryRow("SELECT status FROM moderator_requests WHERE user_id=?", actor).Scan(&status)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if status == "pending" {
			return ErrConflict
		}
		result, err = tx.Exec(`INSERT INTO moderator_requests(user_id) VALUES(?) ON CONFLICT(user_id)
			DO UPDATE SET status='pending',reply='',created_at=CURRENT_TIMESTAMP`, actor)
	case "accept-request", "decline-request":
		if !admin {
			return ErrForbidden
		}
		status := "declined"
		if action == "accept-request" {
			status = "accepted"
			result, err = tx.Exec("UPDATE users SET role='moderator' WHERE id=? AND role='user'", id)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n == 0 {
				return ErrConflict
			}
		}
		result, err = tx.Exec("UPDATE moderator_requests SET status=?,reply=? WHERE user_id=? AND status='pending'", status, text, id)
	case "set-role":
		if !admin {
			return ErrForbidden
		}
		if text != "moderator" && text != "user" {
			return ErrInvalid
		}
		result, err = tx.Exec("UPDATE users SET role=? WHERE id=? AND role!='admin' AND role!=?", text, id, text)
		if err == nil && text == "moderator" {
			_, err = tx.Exec("UPDATE moderator_requests SET status='accepted',reply='Your moderator request was accepted.' WHERE user_id=? AND status='pending'", id)
		}
	case "approve-post", "approve-comment", "reject-comment":
		if !staff {
			return ErrForbidden
		}
		if action == "approve-post" {
			result, err = tx.Exec("UPDATE posts SET status='approved' WHERE id=? AND status='pending'", id)
		}
		if action == "approve-comment" {
			result, err = tx.Exec("UPDATE comments SET status='approved' WHERE id=? AND status='pending' AND post_id IN (SELECT id FROM posts WHERE status='approved')", id)
		}
		if action == "reject-comment" {
			result, err = tx.Exec("DELETE FROM comments WHERE id=? AND status='pending'", id)
		}
	case "delete-post":
		result, err = tx.Exec("DELETE FROM posts WHERE id=? AND (user_id=? OR ?)", id, actor, staff)
		if err == nil {
			n, affectedErr := result.RowsAffected()
			if affectedErr != nil {
				return affectedErr
			}
			if n == 0 {
				var exists bool
				if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id=?)", id).Scan(&exists); err != nil {
					return err
				}
				if exists {
					return ErrForbidden
				}
				return ErrNotFound
			}
		}
	case "delete-comment":
		if !admin {
			return ErrForbidden
		}
		result, err = tx.Exec("DELETE FROM comments WHERE id=?", id)
	case "report":
		if !staff {
			return ErrForbidden
		}
		if text == "" || (reason != "irrelevant" && reason != "obscene" && reason != "illegal" && reason != "insulting") {
			return ErrInvalid
		}
		result, err = tx.Exec("INSERT INTO moderation_reports(post_id,reporter_id,title,reason,message) SELECT id,?,title,?,? FROM posts WHERE id=?", actor, reason, text, id)
	case "reply-report":
		if !admin {
			return ErrForbidden
		}
		if text == "" {
			return ErrInvalid
		}
		result, err = tx.Exec("UPDATE moderation_reports SET reply=? WHERE id=? AND reply=''", text, id)
	case "create-category":
		if !admin {
			return ErrForbidden
		}
		if text == "" || len(text) > 80 || strings.Contains(text, ",") {
			return ErrInvalid
		}
		result, err = tx.Exec("INSERT INTO categories(name) VALUES(?)", text)
	case "delete-category":
		if !admin {
			return ErrForbidden
		}
		var used int
		if err := tx.QueryRow("SELECT COUNT(*) FROM post_categories WHERE category_id=?", id).Scan(&used); err != nil {
			return err
		}
		if used != 0 {
			return ErrConflict
		}
		result, err = tx.Exec("DELETE FROM categories WHERE id=?", id)
	default:
		return ErrInvalid
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConflict
	}
	return tx.Commit()
}

func ModerationData(user *models.User) (models.ModerationPage, error) {
	data := models.ModerationPage{User: user, Staff: user.Role == "moderator" || user.Role == "admin", Admin: user.Role == "admin"}
	rows, err := DBInstance.DB.Query(`SELECT r.user_id,u.username,r.status,r.reply FROM moderator_requests r JOIN users u ON u.id=r.user_id WHERE ? OR r.user_id=? ORDER BY r.created_at`, data.Admin, user.ID)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.RoleRequest
		if err := rows.Scan(&item.UserID, &item.Username, &item.Status, &item.Reply); err != nil {
			rows.Close()
			return data, err
		}
		data.Requests = append(data.Requests, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return data, err
	}
	rows, err = DBInstance.DB.Query(`SELECT r.id,COALESCE(r.post_id,0),u.username,r.title,r.reason,r.message,r.reply FROM moderation_reports r JOIN users u ON u.id=r.reporter_id WHERE ? OR r.reporter_id=? ORDER BY r.id DESC`, data.Admin, user.ID)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.ModerationReport
		if err := rows.Scan(&item.ID, &item.PostID, &item.Reporter, &item.Title, &item.Reason, &item.Message, &item.Reply); err != nil {
			rows.Close()
			return data, err
		}
		data.Reports = append(data.Reports, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return data, err
	}
	if data.Staff {
		rows, err = DBInstance.DB.Query(`SELECT 'post',p.id,p.id,p.title,p.content,u.username FROM posts p JOIN users u ON u.id=p.user_id WHERE p.status='pending'
		UNION ALL SELECT 'comment',c.id,c.post_id,p.title,c.content,u.username FROM comments c JOIN users u ON u.id=c.user_id JOIN posts p ON p.id=c.post_id WHERE c.status='pending'`)
		if err != nil {
			return data, err
		}
		for rows.Next() {
			var item models.PendingContent
			if err := rows.Scan(&item.Kind, &item.ID, &item.PostID, &item.Title, &item.Content, &item.Username); err != nil {
				rows.Close()
				return data, err
			}
			data.Pending = append(data.Pending, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return data, err
		}
	}
	if data.Admin {
		rows, err = DBInstance.DB.Query("SELECT id,username,role FROM users ORDER BY username")
		if err != nil {
			return data, err
		}
		for rows.Next() {
			var user models.User
			if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
				rows.Close()
				return data, err
			}
			data.Users = append(data.Users, user)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return data, err
		}
		data.Categories, err = GetAllCategories()
	}
	return data, err
}

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
