package middleware

import (
	"time"
	"userservice/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
)

func RateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        int(config.MAXReqPerUser),
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			user := c.Locals("user")
			if user != nil {
				claims := user.(*jwt.Token).Claims.(jwt.MapClaims)
				return claims["id"].(string)
			}
			return c.IP()
		},
	})
}
