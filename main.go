package main

import (
	"github.com/gofiber/fiber/v2"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	app := fiber.New()

	// GET /
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, Fiber!")
	})

	// GET /hello/:name
	app.Get("/hello/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		return c.JSON(fiber.Map{
			"message": "Hello, " + name,
		})
	})

	// POST /user
	app.Post("/user", func(c *fiber.Ctx) error {
		var user User

		if err := c.BodyParser(&user); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}

		return c.JSON(fiber.Map{
			"message": "User created",
			"user":    user,
		})
	})

	// Запуск сервера
	app.Listen(":5000")
}