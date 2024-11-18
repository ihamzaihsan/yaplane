package Forum

import (
	"fmt"
	"strings"
	models "forum/models"
)

func GetUserPosts(userID string) ([]models.Post, error) {
	var posts []models.Post

	query := `
		SELECT p.id, p.title, p.content, GROUP_CONCAT(DISTINCT c.name) as categories, p.created_at, u.username, p.image_path,
		COALESCE(COUNT(DISTINCT CASE WHEN l.is_like = 1 THEN l.id ELSE NULL END), 0) AS likes,
		COALESCE(COUNT(DISTINCT CASE WHEN l.is_like = 0 THEN l.id ELSE NULL END), 0) AS dislikes
		FROM posts p
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN likes l ON p.id = l.post_id
		LEFT JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN categories c ON pc.category_id = c.id
		WHERE p.user_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`
	rows, err := DBInstance.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching posts by user: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var categories string
		var username string
		var likes, dislikes int


		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &categories, &post.CreatedAt, &username, &post.ImagePath, &likes, &dislikes); err != nil {
			return nil, fmt.Errorf("error scanning post: %v", err)
		}

	
		post.Categories = strings.Split(categories, ",")
		post.Username = username
		post.Likes = likes
		post.Dislikes = dislikes


		posts = append(posts, post)
	}


	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error with row iteration: %v", err)
	}

	return posts, nil
}
