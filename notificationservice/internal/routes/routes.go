package routes

import (
	"notificationservice/internal/websocket"

	"github.com/gofiber/fiber/v2"
	ws "github.com/gofiber/websocket/v2"
)

func Setup(app *fiber.App, manager *websocket.WebSocketManager) {
	app.Use("/ws", manager.HandleConnections)
	app.Get("/ws", ws.New(manager.WebSocket))
}
