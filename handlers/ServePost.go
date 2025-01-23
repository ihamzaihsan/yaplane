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
	const maxUploadSize = 20 << 20 // 20 MB

	// Check session
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("static/createPosts.html")
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}
		data, err := postFormData(map[string]interface{}{})
		if err != nil {
			moderationError(w, err)
			return
		}
		tmpl.Execute(w, data)
		return
	}

	if r.Method == http.MethodPost {
		userEmail, _ := database.GetEmailFromSession(cookie.Value)
		var userID int
		var role string
		err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		err = r.ParseMultipartForm(maxUploadSize)
		if err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				errMsg := "Invalid input: File size exceeds the 20 MB limit."
				tmpl, tmplErr := template.ParseFiles("static/createPosts.html")
				if tmplErr != nil {
					handleError.ServeError(w, r, http.StatusInternalServerError)
					return
				}
				executePostForm(w, r, tmpl, map[string]interface{}{
					"Error":      errMsg,
					"Title":      strings.TrimSpace(r.FormValue("title")),
					"Content":    strings.TrimSpace(r.FormValue("content")),
					"Categories": r.Form["category_ids[]"],
				})
				return
			}
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		file, handler, err := r.FormFile("image")
		var imagePath string
		if err == nil && file != nil {
			defer file.Close()

			if err := os.MkdirAll("static/uploads", 0755); err != nil {
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}

			buffer := make([]byte, 512)
			_, err := file.Read(buffer)
			if err != nil {
				handleError.ServeError(w, r, http.StatusInternalServerError)
				return
			}
			file.Seek(0, io.SeekStart) // Reset file pointer

			mimeType := http.DetectContentType(buffer)
			if !strings.HasPrefix(mimeType, "image/") {
				http.Error(w, "Invalid file type. Please upload an image.", http.StatusBadRequest)
				return
			}

			// Save file
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

		// Title, content, and category validations (same as your code)
		title := strings.TrimSpace(r.FormValue("title"))
		content := strings.TrimSpace(r.FormValue("content"))
		categoryIDs := r.Form["category_ids[]"]

		// Validate inputs (same as your code)
		tmpl, err := template.ParseFiles("static/createPosts.html")
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}

		if title == "" || content == "" {
			errMsg := "Invalid input: Title and content cannot be empty or just spaces."
			executePostForm(w, r, tmpl, map[string]interface{}{
				"Error":      errMsg,
				"Title":      title,
				"Content":    content,
				"Categories": categoryIDs,
			})
			return
		}

		if len(categoryIDs) == 0 {
			errMsg := "Invalid input: Please select at least one category."
			executePostForm(w, r, tmpl, map[string]interface{}{
				"Error":      errMsg,
				"Title":      title,
				"Content":    content,
				"Categories": categoryIDs,
			})
			return
		}

		// Save post and categories (same as your code)
		status := "approved"
		tx, err := database.DBInstance.DB.Begin()
		if err != nil {
			handleError.ServeError(w, r, http.StatusInternalServerError)
			return
		}
		if err := tx.QueryRow("SELECT role FROM users WHERE id=?", userID).Scan(&role); err != nil {
			tx.Rollback()
			moderationError(w, err)
			return
		}
		status = database.SubmissionStatus(role)
		result, err := tx.Exec(
			"INSERT INTO posts (title, content, image_path, user_id, status) VALUES (?, ?, ?, ?, ?)",
			title, content, imagePath, userID, status,
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

		target := "/"
		if status == "pending" {
			target = "/post/myPosts"
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}
