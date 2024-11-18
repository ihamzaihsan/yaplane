package Forum

import (
	models "forum/models"
	"log"
	"strings"
)

func GetPostDetails(postID string) (models.Post, []models.Comment, int, int) {
	var post models.Post
	var comments []models.Comment
	var likes, dislikes int
	var categoriesStr string


	err := DBInstance.DB.QueryRow(`
        SELECT p.id, p.title, p.content, u.username, p.created_at, p.image_path,
               GROUP_CONCAT(c.name) as categories
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN post_categories pc ON p.id = pc.post_id
        LEFT JOIN categories c ON pc.category_id = c.id
        WHERE p.id = ?
        GROUP BY p.id
    `, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.CreatedAt, &post.ImagePath, &categoriesStr)
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
        SELECT c.id, c.content, u.username, c.created_at 
        FROM comments c 
        JOIN users u ON c.user_id = u.id 
        WHERE c.post_id = ? 
        ORDER BY c.created_at DESC 
    `, postID)
	if err != nil {
		log.Printf("Error fetching comments: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var comment models.Comment
			err := rows.Scan(&comment.ID, &comment.Content, &comment.Username, &comment.CreatedAt)
			if err != nil {
				log.Printf("Error scanning comment: %v", err)
			} else {
			
				comment.Likes = getLikesCount(comment.ID)
				comment.Dislikes = getDislikesCount(comment.ID)

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

func getLikesCount(commentID int) int {
	var count int
	err := DBInstance.DB.QueryRow("SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = true", commentID).Scan(&count)
	if err != nil {
		log.Printf("Error counting likes for comment %d: %v", commentID, err)
	}
	return count
}

func getDislikesCount(commentID int) int {
	var count int
	err := DBInstance.DB.QueryRow("SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = false", commentID).Scan(&count)
	if err != nil {
		log.Printf("Error counting dislikes for comment %d: %v", commentID, err)
	}
	return count
}
