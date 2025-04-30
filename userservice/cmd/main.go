package main

import (
	"userservice/config"
	"userservice/database"
	"userservice/internal/controllers"
	"userservice/internal/kafka"
	"userservice/internal/middleware"
	"userservice/internal/routes"
	"userservice/internal/utils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	producer := kafka.NewProducer(logger)
	defer producer.Close()
	blacklist := utils.NewBlacklist(config.RedisAddr,logger)
	controllers.InitAuthController(config.GRPCAddress, logger, producer, blacklist)
	middleware.InitMiddleware(producer, logger)

	app := fiber.New()

	database.Init(logger)

	routes.Setup(app)

	logger.Info("Server started on port " + config.HTTPPort)
	if err := app.Listen(config.HTTPPort); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
