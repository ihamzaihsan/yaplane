package handlers

import (
	"database/sql"
	"fmt"
	handleError "forum/Error"
	database "forum/database"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func ServePost(w http.ResponseWriter, r *http.Request) {
	
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}


	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("static//templates/createPosts.html")
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
		}
		return
	}
	
	if r.Method == http.MethodPost {
	
		userEmail, _ := database.GetEmailFromSession(cookie.Value)
		var userID int
		err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}


		err = r.ParseMultipartForm(10 << 20) 
		if err != nil {
			log.Println("Error parsing form:", err)
		}


		file, handler, err := r.FormFile("image")
		var imagePath string
		if err == nil && file != nil {
			defer file.Close()

		
			if err := os.MkdirAll("static/uploads", 0755); err != nil {
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}

			
			filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
			imagePath = "uploads/" + filename

		
			dst, err := os.Create("static/" + imagePath)
			if err != nil {
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			
			if _, err := io.Copy(dst, file); err != nil {
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}
		}

	
		title := strings.TrimSpace(r.FormValue("title"))
		content := strings.TrimSpace(r.FormValue("content"))
		categoryIDs := r.Form["category_ids[]"]

	
		tmpl, err := template.ParseFiles("static//templates/createPosts.html")
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		if title == "" || content == "" {
		
			errMsg := "Invalid input: Title and content cannot be empty or just spaces."
			tmpl.Execute(w, map[string]interface{}{
				"Error":      errMsg,
				"Title":      title,
				"Content":    content,
				"Categories": categoryIDs, 
			})
			return
		}

		if len(categoryIDs) == 0 {
		
			errMsg := "Invalid input: Please select at least one category."
			tmpl.Execute(w, map[string]interface{}{
				"Error":      errMsg,
				"Title":      title,
				"Content":    content,
				"Categories": categoryIDs, 
			})
			return
		}

	
		tx, err := database.DBInstance.DB.Begin()
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		
		result, err := tx.Exec(
			"INSERT INTO posts (title, content, image_path, user_id) VALUES (?, ?, ?, ?)",
			title, content, imagePath, userID,
		)
		if err != nil {
			tx.Rollback()
			log.Println("Error inserting post:", err)
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		postID, err := result.LastInsertId()
		if err != nil {
			tx.Rollback()
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		
		for _, categoryID := range categoryIDs {
			_, err = tx.Exec(
				"INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)",
				postID, categoryID,
			)
			if err != nil {
				tx.Rollback()
				log.Println("Error associating category with post:", err)
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}
		}

		
		if err = tx.Commit(); err != nil {
			tx.Rollback()
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

	
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
