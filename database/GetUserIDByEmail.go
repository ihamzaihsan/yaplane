package Forum

import (
	"database/sql"
	"fmt"
)


func GetUserIDByEmail(email string) (string, error) {
	var userID string

	
	query := "SELECT id FROM users WHERE email = ?"


	err := DBInstance.DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			
			return "", fmt.Errorf("no user found with the provided email")
		}

		return "", fmt.Errorf("error querying the database: %v", err)
	}

	return userID, nil
}
