package middleware

import (
	"time"
	"userservice/config"
	"userservice/internal/kafka"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var producer *kafka.Producer
var logger *zap.Logger

func InitMiddleware(p *kafka.Producer, l *zap.Logger) {
	producer = p
	logger = l
}

func JWTProtected() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   []byte(config.JWTSecret),
		ContextKey:   "user",
		ErrorHandler: jwtError,
		TokenLookup:  "header:Authorization",
		AuthScheme:   "Bearer",
	})
}

func jwtError(c *fiber.Ctx, err error) error {
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return nil
}

func AdminOnly(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userId := claims["id"].(string)
	role := claims["role"].(string)

	if role != "admin" {
		logEvent := map[string]interface{}{
			"message": "Forbidden access attempt by non-admin user",
			"userId":  userId,
			"role":    role,
			"time":    time.Now().Format(time.RFC3339),
		}
		if logErr := producer.SendMessageToLogsTopic(userId, logEvent); logErr != nil {
			logger.Error("Failed to send log event", zap.Error(logErr))
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	return c.Next()
}

func UserOnly(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userId := claims["id"].(string)
	role := claims["role"].(string)

	if role != "user" {
		logEvent := map[string]interface{}{
			"message": "Forbidden access attempt by admin user",
			"userId":  userId,
			"role":    role,
			"time":    time.Now().Format(time.RFC3339),
		}
		if logErr := producer.SendMessageToLogsTopic(userId, logEvent); logErr != nil {
			logger.Error("Failed to send log event", zap.Error(logErr))
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	return c.Next()
}

// func BlacklistCheckMiddleware() fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		token := c.Get("Authorization")
// 		if token == "" {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or malformed JWT"})
// 		}

// 		// Remove "Bearer " prefix
// 		token = token[7:]

// 		if utils.IsBlacklisted(token) {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token is blacklisted"})
// 		}

// 		return c.Next()
// 	}
// }
