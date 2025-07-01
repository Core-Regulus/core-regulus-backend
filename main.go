package main

import (
	"core-regulus-backend/internal/db"
	"core-regulus-backend/internal/calendar"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "POST, OPTIONS",
	}))
	db.Connect()
	calendar.InitRoutes(app)

	app.Listen(":5000")
}