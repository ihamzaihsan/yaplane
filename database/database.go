package Forum

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DataBase struct {
	DB *sql.DB
}

var DBInstance DataBase

func InitDB() error {
	var err error
	DBInstance.DB, err = sql.Open("sqlite3", "Forum.db?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	// A single connection serializes SQLite writes and session replacement transactions.
	DBInstance.DB.SetMaxOpenConns(1)
	err = DBInstance.DB.Ping()
	if err != nil {
		return fmt.Errorf("error pinging database: %v", err)
	}

	_, err = DBInstance.DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("error enabling foreign keys: %v", err)
	}

	err = CreateTables(DBInstance.DB)
	if err != nil {
		return fmt.Errorf("error creating tables: %v", err)
	}
	AddDefaultCategories(DBInstance.DB)

	return nil
}

func CreateTables(db *sql.DB) error {

	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	createCategoriesTable := `
    CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE
    );`

	if _, err := db.Exec(createCategoriesTable); err != nil {
		return fmt.Errorf("failed to create categories table: %v", err)
	}

	createPostsTable := `
    CREATE TABLE IF NOT EXISTS posts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        image_path TEXT,
        user_id INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createPostsTable); err != nil {
		return fmt.Errorf("failed to create posts table: %v", err)
	}

	createPostCategoriesTable := `
      CREATE TABLE IF NOT EXISTS post_categories (
          post_id INTEGER,
          category_id INTEGER,
          PRIMARY KEY (post_id, category_id),
          FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
          FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE
      );`

	if _, err := db.Exec(createPostCategoriesTable); err != nil {
		return fmt.Errorf("failed to create post_categories table: %v", err)
	}

	createCommentsTable := `
    CREATE TABLE IF NOT EXISTS comments (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        content TEXT NOT NULL,
        user_id INTEGER NOT NULL,
        post_id INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createCommentsTable); err != nil {
		return fmt.Errorf("failed to create comments table: %v", err)
	}

	createLikesTable := `
    CREATE TABLE IF NOT EXISTS likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    post_id INTEGER,
    comment_id INTEGER,
    is_like BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE,
    CONSTRAINT unique_like UNIQUE (user_id, post_id, comment_id)
  );`

	if _, err := db.Exec(createLikesTable); err != nil {
		return fmt.Errorf("failed to create likes table: %v", err)
	}

	createSessionTable := `
    CREATE TABLE IF NOT EXISTS sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_token TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL,
        expires_at DATETIME NOT NULL,
        FOREIGN KEY (email) REFERENCES users (email) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createSessionTable); err != nil {
		return fmt.Errorf("failed to create sessions table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS oauth_identities (
		provider TEXT NOT NULL CHECK(provider IN ('google','github')),
		subject TEXT NOT NULL,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY(provider,subject), UNIQUE(provider,user_id)
	)`); err != nil {
		return fmt.Errorf("failed to create OAuth identities table: %w", err)
	}

	return nil
}
