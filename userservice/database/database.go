package database

import (
	"database/sql"
	"fmt"
	"userservice/config"
	"userservice/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var DB *sql.DB
var logger *zap.Logger

func Init(l *zap.Logger) {
	var err error
	logger = l
	DB, err = sql.Open("sqlite3", config.UserDBPath)
	if err != nil {
		logger.Fatal("Connecting to DB failed with error", zap.Error(err))
	}

	createUserTable()
}

func createUserTable() {
	query := `
    CREATE TABLE IF NOT EXISTS users (id TEXT NOT NULL PRIMARY KEY,email TEXT NOT NULL,password TEXT NOT NULL,
        role TEXT NOT NULL
    );
    `
	_, err := DB.Exec(query)
	if err != nil {
		logger.Fatal("Failed to create users table", zap.Error(err))
	}
}

func CreateUser(user models.User) error {
	_, err := DB.Exec("INSERT INTO users (id, email, password, role) VALUES (?, ?, ?, ?)", user.ID, user.Email, user.Password, user.Role)
	return err
}

func GetUserById(userId string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT id, email, role FROM users WHERE id = '%s'`, userId)
	row := DB.QueryRow(query)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.Role)
	if err == sql.ErrNoRows {
		logger.Error("User not found", zap.String("userId", userId))
		return nil, nil
	} else if err != nil {
		logger.Error("Failed to get user", zap.String("userId", userId), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

func GetUsersWithRole(role string) ([]models.User, error) {
	rows, err := DB.Query("SELECT id, email, role FROM users WHERE role = ?", role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
