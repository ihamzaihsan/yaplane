package Forum

import (
	models "forum/models"
	"log"
	"strings"
)

func GetPostDetails(postID string, viewer *models.User) (models.Post, []models.Comment, int, int) {
	actor, staff := 0, false
	if viewer != nil {
		actor = viewer.ID
		staff = viewer.Role == "moderator" || viewer.Role == "admin"
	}
	var post models.Post
	var comments []models.Comment
	var likes, dislikes int
	var categoriesStr string

	err := DBInstance.DB.QueryRow(`
        SELECT p.id, p.title, p.content, u.username, p.created_at, p.image_path,
               COALESCE(GROUP_CONCAT(c.name),'') as categories, p.status
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN post_categories pc ON p.id = pc.post_id
        LEFT JOIN categories c ON pc.category_id = c.id
        WHERE p.id = ? AND (p.status='approved' OR p.user_id=? OR ?)
        GROUP BY p.id
    `, postID, actor, staff).Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.CreatedAt, &post.ImagePath, &categoriesStr, &post.Status)
	if err != nil {
		log.Printf("Error fetching post details: %v", err)
		return post, comments, likes, dislikes
	}

	if categoriesStr != "" {
		post.Categories = strings.Split(categoriesStr, ",")
	} else {
		post.Categories = []string{}
	}

	rows, err := DBInstance.DB.Query(`
        SELECT c.id, c.content, u.username, c.created_at, c.status,
               (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 1),
               (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 0)
        FROM comments c 
        JOIN users u ON c.user_id = u.id 
        WHERE c.post_id = ? AND (c.status='approved' OR c.user_id=? OR ?)
        ORDER BY c.created_at DESC 
    `, postID, actor, staff)
	if err != nil {
		log.Printf("Error fetching comments: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var comment models.Comment
			err := rows.Scan(&comment.ID, &comment.Content, &comment.Username, &comment.CreatedAt, &comment.Status, &comment.Likes, &comment.Dislikes)
			if err != nil {
				log.Printf("Error scanning comment: %v", err)
			} else {

				comments = append(comments, comment)
			}
		}
	}

	err = DBInstance.DB.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = true", postID).Scan(&likes)
	if err != nil {
		log.Printf("Error counting likes: %v", err)
	}

	err = DBInstance.DB.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = false", postID).Scan(&dislikes)
	if err != nil {
		log.Printf("Error counting dislikes: %v", err)
	}

	return post, comments, likes, dislikes
}
