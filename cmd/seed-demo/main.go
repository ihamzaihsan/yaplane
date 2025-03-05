package main

import (
	"errors"
	"flag"
	"fmt"
	database "forum/database"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Demo seed:", err)
		os.Exit(1)
	}
}

func run() error {
	defaultPath := os.Getenv("DATABASE_PATH")
	if defaultPath == "" {
		defaultPath = "data/demo.db"
	}
	path := flag.String("database", defaultPath, "new SQLite file for fictional demo data")
	flag.Parse()
	if flag.NArg() != 0 || strings.TrimSpace(*path) == "" {
		return errors.New("provide a database path with -database")
	}
	abs, err := filepath.Abs(*path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return err
	}
	// Reserve a new file exclusively before initialization can migrate any data.
	file, err := os.OpenFile(abs, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("refusing to change existing database %s; choose a new path", abs)
	}
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	db, err := database.Open(abs)
	if err != nil {
		return err
	}
	defer db.Close()
	accounts, err := database.SeedDemo(db)
	if err != nil {
		return err
	}
	fmt.Printf("Created demo database: %s\n5 accounts, 15 posts, 30 replies, 90 reactions.\n\n", abs)
	fmt.Println("ID  Username       Email")
	for _, account := range accounts {
		fmt.Printf("%-3d %-14s %s\n", account.ID, account.Username, account.Email)
	}
	fmt.Printf("\nPassword for every demo account: %s\nSign in using the email address. All demo accounts are ordinary members.\n", database.DemoPassword)
	return nil
}
