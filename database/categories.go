package Forum

import (
	"database/sql"
	models "forum/models"
	"net/http"
	"strings"
)

func AssociateCategoryWithPost(postID, categoryID int) error {
	_, err := DBInstance.DB.Exec("UPDATE posts SET category_id = ? WHERE id = ?", categoryID, postID)
	return err
}
func GetPostsByCategories(selectedCategories []string) ([]models.Post, error) {

	query := `
		SELECT p.id, p.title, p.content, u.username, p.created_at, p.image_path,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 1) AS likes,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 0) AS dislikes,
		       GROUP_CONCAT(DISTINCT c.name) AS categories
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN categories c ON pc.category_id = c.id
		WHERE p.id IN (
			SELECT p.id
			FROM posts p
			LEFT JOIN post_categories pc ON p.id = pc.post_id
			LEFT JOIN categories c ON pc.category_id = c.id
			WHERE c.name IN (?` + strings.Repeat(",?", len(selectedCategories)-1) + `)
		)
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	args := make([]interface{}, len(selectedCategories))
	for i, category := range selectedCategories {
		args[i] = category
	}

	rows, err := DBInstance.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var categoriesString string
		err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.ImagePath, &p.Likes, &p.Dislikes, &categoriesString)
		if err != nil {
			return nil, err
		}

		p.Categories = strings.Split(categoriesString, ",")
		posts = append(posts, p)
	}

	return posts, nil
}

func AddDefaultCategories(db *sql.DB) error {

	categories := []string{"science", "technology", "art", "sport", "games"}

	stmt, err := db.Prepare("INSERT INTO categories (name) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, category := range categories {
		_, err := stmt.Exec(category)
		if err != nil {

			return err
		}
	}
	return nil
}

func ValidateCategoriesPath(r *http.Request) bool {
	validCategories := []string{"technology", "science", "art", "sport", "games"}

	for key := range r.URL.Query() {
		if key != "categories" {
			return false
		}
	}

	categories := r.URL.Query()["categories"]
	for _, cat := range categories {
		isValid := false
		for _, validCat := range validCategories {
			if cat == validCat {
				isValid = true
				break
			}
		}
		if !isValid {
			return false
		}
	}
	return true
}
