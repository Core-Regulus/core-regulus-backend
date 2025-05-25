package main

import (
	"github.com/gofiber/fiber/v2"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
	{ID: 2, Name: "Bob", Email: "bob@example.com"},
}

func main() {
	app := fiber.New()

	// Получить всех пользователей
	app.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(users)
	})

	// Получить одного пользователя по ID
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(400).SendString("Invalid ID")
		}
		for _, u := range users {
			if u.ID == id {
				return c.JSON(u)
			}
		}
		return c.Status(404).SendString("User not found")
	})

	// Создать пользователя
	app.Post("/users", func(c *fiber.Ctx) error {
		var user User
		if err := c.BodyParser(&user); err != nil {
			return c.Status(400).SendString("Bad Request")
		}
		user.ID = len(users) + 1
		users = append(users, user)
		return c.Status(201).JSON(user)
	})

	// Обновить пользователя
	app.Put("/users/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(400).SendString("Invalid ID")
		}

		var update User
		if err := c.BodyParser(&update); err != nil {
			return c.Status(400).SendString("Bad Request")
		}

		for i, u := range users {
			if u.ID == id {
				users[i].Name = update.Name
				users[i].Email = update.Email
				return c.JSON(users[i])
			}
		}

		return c.Status(404).SendString("User not found")
	})

	// Удалить пользователя
	app.Delete("/users/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(400).SendString("Invalid ID")
		}

		for i, u := range users {
			if u.ID == id {
				users = append(users[:i], users[i+1:]...)
				return c.SendStatus(204)
			}
		}

		return c.Status(404).SendString("User not found")
	})

	app.Listen(":9001")
}
