package Forum

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

func ValidateUser(email, password string) bool {
	var hashedPassword string
	err := DBInstance.DB.QueryRow("SELECT password FROM users WHERE email = ?", email).Scan(&hashedPassword)

	if err != nil {
		log.Printf("Error querying user: %v", err) 
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		log.Printf("Password comparison failed for email: %s - Error: %v", email, err) 
		return false
	}

	return true
}
