package main

import (
	"core-regulus-backend/internal/db"
	"core-regulus-backend/internal/calendar"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	db.Connect()
	calendar.InitRoutes(app)

	app.Listen(":5000")
}