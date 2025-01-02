package handlers

import (
	"errors"
	database "forum/database"
	"net/http"
	"strings"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func ServeRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		authPage(w, r, "register", 200, map[string]interface{}{})
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", 400)
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))
	confirm := strings.TrimSpace(r.FormValue("confirm_password"))
	fail := func(status int, message string) {
		authPage(w, r, "register", status, map[string]interface{}{"Error": message})
	}
	if username == "" || email == "" || password == "" || confirm == "" {
		fail(400, "All fields are required or can't be empty spaces.")
		return
	}
	if !validEmail(email) {
		fail(400, "Enter a valid email address.")
		return
	}
	if len(password) > 72 {
		fail(400, "Password must be no longer than 72 bytes.")
		return
	}
	if password != confirm {
		fail(400, "Passwords do not match.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error securing password", 500)
		return
	}
	_, err = database.DBInstance.DB.Exec("INSERT INTO users (username,email,password) VALUES (?,?,?)", username, email, hash)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			fail(409, "Username or email already exists.")
		} else {
			http.Error(w, "Unable to create the account", 500)
		}
		return
	}
	if err := startSession(w, email); err != nil {
		http.Error(w, "Unable to create a session", 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
