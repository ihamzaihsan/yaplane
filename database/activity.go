package Forum

import (
	"database/sql"
	"forum/models"
)

func migrateActivity(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		actor_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
		kind TEXT NOT NULL CHECK(kind IN ('liked','disliked','commented')),
		is_read BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS notification_inbox ON notifications(recipient_id,is_read,id);
	CREATE TRIGGER IF NOT EXISTS notify_post_reaction AFTER INSERT ON likes WHEN NEW.post_id IS NOT NULL BEGIN
		INSERT INTO notifications(recipient_id,actor_id,post_id,kind)
		SELECT user_id,NEW.user_id,id,CASE WHEN NEW.is_like THEN 'liked' ELSE 'disliked' END FROM posts
		WHERE id=NEW.post_id AND status='approved' AND user_id!=NEW.user_id;
	END;
	CREATE TRIGGER IF NOT EXISTS notify_changed_reaction AFTER UPDATE OF is_like ON likes
	WHEN NEW.post_id IS NOT NULL AND OLD.is_like!=NEW.is_like BEGIN
		INSERT INTO notifications(recipient_id,actor_id,post_id,kind)
		SELECT user_id,NEW.user_id,id,CASE WHEN NEW.is_like THEN 'liked' ELSE 'disliked' END FROM posts
		WHERE id=NEW.post_id AND status='approved' AND user_id!=NEW.user_id;
	END;
	CREATE TRIGGER IF NOT EXISTS notify_post_comment AFTER INSERT ON comments WHEN NEW.status='approved' BEGIN
		INSERT INTO notifications(recipient_id,actor_id,post_id,comment_id,kind)
		SELECT user_id,NEW.user_id,id,NEW.id,'commented' FROM posts WHERE id=NEW.post_id AND status='approved' AND user_id!=NEW.user_id;
	END;
	CREATE TRIGGER IF NOT EXISTS notify_approved_comment AFTER UPDATE OF status ON comments
	WHEN OLD.status='pending' AND NEW.status='approved' BEGIN
		INSERT INTO notifications(recipient_id,actor_id,post_id,comment_id,kind)
		SELECT user_id,NEW.user_id,id,NEW.id,'commented' FROM posts WHERE id=NEW.post_id AND status='approved' AND user_id!=NEW.user_id;
	END;
	CREATE TRIGGER IF NOT EXISTS hide_pending_comment_notifications AFTER UPDATE OF status ON comments
	WHEN NEW.status='pending' BEGIN DELETE FROM notifications WHERE comment_id=NEW.id; END;`)
	return err
}

func NotificationCounts(userID int) (unread, latest int, err error) {
	err = DBInstance.DB.QueryRow("SELECT COALESCE(SUM(NOT is_read),0),COALESCE(MAX(id),0) FROM notifications WHERE recipient_id=?", userID).Scan(&unread, &latest)
	return
}

func Notifications(userID int) (models.NotificationPage, error) {
	var data models.NotificationPage
	var err error
	data.Unread, data.Latest, err = NotificationCounts(userID)
	if err != nil {
		return data, err
	}
	rows, err := DBInstance.DB.Query(`SELECT n.id,n.post_id,u.username,p.title,n.kind,n.is_read,n.created_at
		FROM notifications n JOIN users u ON u.id=n.actor_id JOIN posts p ON p.id=n.post_id
		WHERE n.recipient_id=? ORDER BY n.id DESC LIMIT 100`, userID)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.Notification
		if err := rows.Scan(&item.ID, &item.PostID, &item.Actor, &item.Title, &item.Kind, &item.Read, &item.CreatedAt); err != nil {
			return data, err
		}
		data.Items = append(data.Items, item)
	}
	return data, rows.Err()
}

func ReadNotifications(userID int) error {
	_, err := DBInstance.DB.Exec("UPDATE notifications SET is_read=1 WHERE recipient_id=? AND is_read=0", userID)
	return err
}

// Serialize reaction replacement with its notification; retries of the same choice
// keep one reaction and do not create duplicate notifications.
