package jwtpkg

import (
	"time"
	"userservice/config"
	"userservice/internal/models"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateJWT(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret))
}
