package Forum

import (
	"database/sql"
	models "forum/models"
	"net/http"
	"strings"
)

func GetAllCategories() ([]models.Category, error) {
	rows, err := DBInstance.DB.Query("SELECT id, name FROM categories ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
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
		WHERE p.status='approved' AND p.id IN (
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

	return posts, rows.Err()
}

func AddDefaultCategories(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS forum_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		return err
	}
	var seeded bool
	if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM forum_settings WHERE key='categories_seeded')").Scan(&seeded); err != nil {
		return err
	}
	if !seeded {
		for _, category := range []string{"science", "technology", "art", "sport", "games"} {
			if _, err := tx.Exec("INSERT OR IGNORE INTO categories(name) VALUES(?)", category); err != nil {
				return err
			}
		}
		if _, err := tx.Exec("INSERT INTO forum_settings VALUES('categories_seeded','true')"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ValidateCategoriesPath(r *http.Request, all []models.Category) bool {
	validCategories := make([]string, 0, len(all))
	for _, category := range all {
		validCategories = append(validCategories, category.Name)
	}

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
