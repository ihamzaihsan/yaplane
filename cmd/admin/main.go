// Assign the first administrator through local database access, never registration.
package main

import (
	"database/sql"
	"flag"
	database "forum/database"
	"log"
	"os"
	"strings"
)

func main() {
	email := flag.String("email", "", "existing account email to promote")
	path := flag.String("db", "Forum.db", "existing SQLite database path")
	flag.Parse()
	if strings.TrimSpace(*email) == "" {
		log.Fatal("Provide -email for an existing registered account")
	}
	info, err := os.Stat(*path)
	if err != nil || !info.Mode().IsRegular() {
		log.Fatal("The database must already exist; start the forum and register an account first")
	}
	db, err := sql.Open("sqlite3", *path+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := database.BootstrapAdmin(db, *email); err != nil {
		log.Fatal(err)
	}
	log.Print("Administrator assigned. Sign in to the forum and open /moderation.")
}
