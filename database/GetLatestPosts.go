package Forum

import (
	models "forum/models"
	"strings"
)

func GetLatestPosts() ([]models.Post, error) {
	query := `
        SELECT p.id, p.title, p.content, u.username, p.created_at, p.image_path,
               (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 1) AS likes,
               (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 0) AS dislikes,
               GROUP_CONCAT(c.name) AS categories
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN post_categories pc ON p.id = pc.post_id
        LEFT JOIN categories c ON pc.category_id = c.id
        GROUP BY p.id
        ORDER BY p.created_at DESC
        LIMIT 10
    `
	rows, err := DBInstance.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var categories string
		err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.ImagePath, &p.Likes, &p.Dislikes, &categories)

		if err != nil {
			return nil, err
		}
		p.Categories = strings.Split(categories, ",")
		posts = append(posts, p)
	}
	return posts, nil
}
