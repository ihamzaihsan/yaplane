package Forum

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// DemoPassword is intentionally public and only for fictional demo accounts.
const DemoPassword = "DemoReview!2026"

type DemoAccount struct {
	ID       int64
	Username string
	Email    string
}

// SeedDemo inserts fictional content atomically and never adds to user data.
func SeedDemo(db *sql.DB) ([]DemoAccount, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRow(`SELECT (SELECT COUNT(*) FROM users) + (SELECT COUNT(*) FROM posts)
		+ (SELECT COUNT(*) FROM comments) + (SELECT COUNT(*) FROM likes)
		+ (SELECT COUNT(*) FROM sessions) + (SELECT COUNT(*) FROM oauth_identities)
		+ (SELECT COUNT(*) FROM moderator_requests) + (SELECT COUNT(*) FROM moderation_reports)
		+ (SELECT COUNT(*) FROM notifications)`).Scan(&count); err != nil {
		return nil, err
	}
	if count != 0 {
		return nil, errors.New("demo seed requires a database without user content")
	}
	insert := func(query string, args ...interface{}) (int64, error) {
		result, err := tx.Exec(query, args...)
		if err != nil {
			return 0, fmt.Errorf("insert demo content: %w", err)
		}
		return result.LastInsertId()
	}
	now := time.Now().UTC().Truncate(time.Second)
	members := []string{"alex_demo", "mia_demo", "sam_demo", "noor_demo", "leo_demo"}
	accounts := make([]DemoAccount, 0, len(members))
	for _, member := range members {
		hash, err := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		email := member + "@example.com"
		id, err := insert(`INSERT INTO users(username,email,password,created_at) VALUES(?,?,?,?)`, member, email, string(hash), now.AddDate(0, 0, -30))
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, DemoAccount{ID: id, Username: member, Email: email})
	}
	posts := []struct {
		category, title, content string
		replies                  [2]string
	}{
		{"technology", "Welcome to the Yaplane Community demo", "This is a fictional community for exploring the project. Try filtering topics, reacting to a discussion, or reviewing your activity and notifications.", [2]string{"The topic filters make it easy to find a discussion.", "I tried posting a reply and the mobile layout too."}},
		{"science", "What sparked your interest in astronomy?", "A clear night and a borrowed telescope got me interested in astronomy. What observation or book first made you curious about space?", [2]string{"Seeing Saturn through a telescope was my starting point.", "A library book about the solar system did it for me."}},
		{"art", "Share your approach to a daily sketch habit", "I am trying short daily sketches instead of waiting for a perfect idea. How do you choose a subject when you only have fifteen minutes?", [2]string{"I draw one ordinary object on my desk each day.", "Repeating the same subject helps me notice progress."}},
		{"sport", "How do you plan a beginner running week?", "I am comparing easy runs, rest days, and a weekend walk for a fictional training plan. What keeps your routine enjoyable and consistent?", [2]string{"Having a friend join makes it easier to stay consistent.", "I like leaving space for rest rather than filling every day."}},
		{"games", "Cooperative games for a relaxed evening", "Our demo game group wants something cooperative with a short setup. Which features make a game welcoming to a new player?", [2]string{"A clear shared goal helps everyone get started.", "I prefer games where experienced players can explain as we play."}},
		{"technology", "Small details that improve a chat interface", "Message history, clear connection status, and confirmation after saving can make chat easier to trust. Which interface details do you notice first?", [2]string{"Keeping a draft after a connection interruption is useful.", "A visible distinction between online presence and saved history helps."}},
		{"science", "A notebook for everyday science questions", "I started collecting questions from everyday observations, like why a window fogs up. What would you put on the first page of your notebook?", [2]string{"I would start with how shadows change through the day.", "My first question would be about the shapes of clouds."}},
		{"art", "Choosing a limited color palette", "For a small poster project, I am experimenting with two main colors and one accent. How do you decide what should stand out?", [2]string{"I sketch the layout in grayscale before choosing colors.", "One accent works well when it highlights the main action."}},
		{"sport", "Your favorite way to follow a team season", "Some fans follow every fixture, while others enjoy a weekly recap. What makes following a team fun for you?", [2]string{"Talking about the match with friends is my favorite part.", "A weekly recap fits my schedule better than every game."}},
		{"games", "What makes a puzzle feel fair?", "I enjoy puzzles where an overlooked clue becomes obvious afterward. How do you distinguish a satisfying challenge from a confusing one?", [2]string{"Consistent rules make a difficult puzzle feel fair.", "I like hints that point me toward a method rather than the answer."}},
		{"technology", "A weekend project with Go and SQLite", "A small forum is a useful way to explore HTTP handlers, relational data, and session authentication together. Which part would you build first?", [2]string{"I would start with creating and listing posts.", "Then I would add comments, reactions, and account activity."}},
		{"science", "Sharing observations from a neighborhood walk", "We are collecting fictional notes about plants, birds, and changing daylight. What details help turn a casual observation into a useful record?", [2]string{"Writing down the time and location helps with comparisons.", "A quick sketch can capture details that words miss."}},
		{"art", "Design feedback that is easy to act on", "I find specific feedback more helpful than a general reaction. How do you explain a design suggestion while leaving room for the creator's intent?", [2]string{"Describe what you noticed and what confused you.", "A concrete example makes the suggestion easier to try."}},
		{"sport", "Building a friendly community tournament", "Our fictional club wants a small tournament where beginners feel welcome. What would you include besides the match schedule?", [2]string{"A clear explanation of the rules before the first match.", "Time for people to meet teammates would help."}},
		{"games", "Games that tell stories through their world", "I like discovering a story through scenery and small interactions. What environmental detail makes a game world memorable for you?", [2]string{"Objects that suggest someone lived there make a difference.", "Changes in music can make a familiar location feel new."}},
	}
	for i, post := range posts {
		author := i % len(accounts)
		created := now.Add(-time.Duration(len(posts)-i) * 12 * time.Hour)
		postID, err := insert(`INSERT INTO posts(title,content,user_id,created_at,image_path) VALUES(?,?,?,?,'')`, post.title, post.content, accounts[author].ID, created)
		if err != nil {
			return nil, err
		}
		categoryResult, err := tx.Exec(`INSERT INTO post_categories(post_id,category_id) SELECT ?,id FROM categories WHERE name=?`, postID, post.category)
		if err != nil {
			return nil, err
		}
		if added, err := categoryResult.RowsAffected(); err != nil || added != 1 {
			return nil, fmt.Errorf("could not assign demo category %q", post.category)
		}
		for j, reply := range post.replies {
			commentID, err := insert(`INSERT INTO comments(content,user_id,post_id,created_at) VALUES(?,?,?,?)`, reply, accounts[(author+j+1)%len(accounts)].ID, postID, created.Add(time.Duration(j+1)*time.Hour))
			if err != nil {
				return nil, err
			}
			if _, err := insert(`INSERT INTO likes(user_id,comment_id,is_like,created_at) VALUES(?,?,?,?)`, accounts[(author+j+3)%len(accounts)].ID, commentID, true, created.Add(3*time.Hour)); err != nil {
				return nil, err
			}
		}
		for j := 1; j < len(accounts); j++ {
			if _, err := insert(`INSERT INTO likes(user_id,post_id,is_like,created_at) VALUES(?,?,?,?)`, accounts[(author+j)%len(accounts)].ID, postID, j <= 1+i%4, created.Add(4*time.Hour)); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return accounts, nil
}
