package Forum

import (
	"database/sql"
	"forum/models"
	"strconv"
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

func Activity(user *models.User) (models.ActivityPage, error) {
	data := models.ActivityPage{User: user}
	var err error
	data.Posts, err = GetUserPosts(strconv.Itoa(user.ID))
	if err != nil {
		return data, err
	}
	staff := user.Role == "moderator" || user.Role == "admin"
	rows, err := DBInstance.DB.Query(`SELECT c.id,c.post_id,c.content,c.status,
		CASE WHEN p.status='approved' OR p.user_id=? OR ? THEN p.title ELSE 'Discussion awaiting review' END,
		(p.status='approved' OR p.user_id=? OR ?)
		FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.user_id=? ORDER BY c.created_at DESC,c.id DESC`, user.ID, staff, user.ID, staff, user.ID)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.ActivityComment
		if err := rows.Scan(&item.ID, &item.PostID, &item.Content, &item.Status, &item.Title, &item.CanView); err != nil {
			rows.Close()
			return data, err
		}
		data.Comments = append(data.Comments, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return data, err
	}
	rows, err = DBInstance.DB.Query(`SELECT p.id,COALESCE(l.comment_id,0),p.title,COALESCE(c.content,''),l.is_like
		FROM likes l LEFT JOIN comments c ON c.id=l.comment_id JOIN posts p ON p.id=COALESCE(l.post_id,c.post_id)
		WHERE l.user_id=? AND p.status='approved' AND (l.comment_id IS NULL OR c.status='approved') ORDER BY l.created_at DESC,l.id DESC`, user.ID)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ActivityReaction
		if err := rows.Scan(&item.PostID, &item.CommentID, &item.Title, &item.Content, &item.IsLike); err != nil {
			return data, err
		}
		data.Reactions = append(data.Reactions, item)
	}
	return data, rows.Err()
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
func RecordReaction(userID, id int, comment, isLike bool) (int, int, error) {
	tx, err := DBInstance.DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	field := "post_id"
	query := "SELECT EXISTS(SELECT 1 FROM posts WHERE id=? AND status='approved')"
	if comment {
		field = "comment_id"
		query = "SELECT EXISTS(SELECT 1 FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.id=? AND c.status='approved' AND p.status='approved')"
	}
	var exists bool
	if err := tx.QueryRow(query, id).Scan(&exists); err != nil {
		return 0, 0, err
	}
	if !exists {
		return 0, 0, ErrNotFound
	}
	var hasReaction bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM likes WHERE user_id=? AND "+field+"=?)", userID, id).Scan(&hasReaction)
	if err != nil {
		return 0, 0, err
	}
	if !hasReaction {
		_, err = tx.Exec("INSERT INTO likes(user_id,"+field+",is_like) VALUES(?,?,?)", userID, id, isLike)
	} else {
		_, err = tx.Exec("UPDATE likes SET is_like=? WHERE user_id=? AND "+field+"=? AND is_like!=?", isLike, userID, id, isLike)
	}
	if err != nil {
		return 0, 0, err
	}
	var likes, dislikes int
	if err := tx.QueryRow("SELECT COALESCE(SUM(is_like=1),0),COALESCE(SUM(is_like=0),0) FROM likes WHERE "+field+"=?", id).Scan(&likes, &dislikes); err != nil {
		return 0, 0, err
	}
	return likes, dislikes, tx.Commit()
}
