package routes

import (
	"userservice/internal/controllers"
	"userservice/internal/middleware"

	"github.com/ansrivas/fiberprometheus/v2"
	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	prometheus := fiberprometheus.New("userservice")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)
	app.Use(middleware.RequestLogger())
	app.Use(middleware.RateLimiter())
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	api.Post("/register", controllers.Register)
	api.Post("/login", controllers.Login)
	api.Post("/logout", middleware.JWTProtected(), controllers.Logout)

	protected := api.Group("/profile")
	protected.Use(middleware.JWTProtected())
	protected.Get("/", controllers.GetProfile)

	admin := api.Group("/admin")
	admin.Use(middleware.JWTProtected(), middleware.AdminOnly)
	admin.Get("/", controllers.GetUsersWithUserRole)
	user := api.Group("/user-data")
	user.Use(middleware.JWTProtected(), middleware.UserOnly)
	user.Get("/:userID", controllers.GetUserByID)
}
