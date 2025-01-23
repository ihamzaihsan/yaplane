package Forum

import (
	"fmt"
	models "forum/models"
	"strings"
)

func GetLikedPostsByUser(userID string) ([]models.Post, error) {
	var posts []models.Post

	query := `
SELECT p.id, p.title, p.content, p.image_path, GROUP_CONCAT(DISTINCT c.name) as categories, p.created_at, u.username,
COALESCE(likes.like_count, 0) AS likes,
COALESCE(dislikes.dislike_count, 0) AS dislikes
FROM posts p
INNER JOIN users u ON p.user_id = u.id
LEFT JOIN post_categories pc ON p.id = pc.post_id
LEFT JOIN categories c ON pc.category_id = c.id
-- Subquery to count likes for each post
LEFT JOIN (
SELECT post_id, COUNT(*) AS like_count
FROM likes
WHERE is_like = 1
GROUP BY post_id
) AS likes ON p.id = likes.post_id
-- Subquery to count dislikes for each post
LEFT JOIN (
SELECT post_id, COUNT(*) AS dislike_count
FROM likes
WHERE is_like = 0
GROUP BY post_id
) AS dislikes ON p.id = dislikes.post_id
WHERE p.status='approved' AND p.id IN (
SELECT post_id FROM likes WHERE user_id = ? AND is_like = 1
)
GROUP BY p.id
ORDER BY p.created_at DESC
`

	rows, err := DBInstance.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching liked posts: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var categories string
		var username string
		var likes, dislikes int

		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.ImagePath, &categories, &post.CreatedAt, &username, &likes, &dislikes); err != nil {
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
