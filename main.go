package main

import (
	"core-regulus-backend/internal/db"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("missing env var: %s", key)
	}
	return val
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
	
	_ = godotenv.Load(".env")
	privateKey := strings.ReplaceAll(mustEnv("SSH_PRIVATE_KEY"), `\n`, "\n")

	sshPort, _ := strconv.Atoi(mustEnv("SSH_PORT"))
	dbPort, _ := strconv.Atoi(mustEnv("DB_PORT"))

	cfg := db.SSHPostgresConfig{
		SSHUser:     mustEnv("SSH_USER"),
		PrivateKeyPEM: privateKey,
		SSHHost:     "deploy.int-t.com",
		SSHPort:     sshPort,
		DBUser:     "postgres",
		DBPassword: "",
		DBName:     "coreregulus",
		DBHost:     "127.0.0.1", 
		DBPort:     dbPort,
	}

	conn, err := db.ConnectViaSSH(cfg)
	if err != nil {
		log.Fatal("Ошибка:", err)
	}
	defer conn.Close()

	var version string
	err = conn.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("PostgreSQL version:", version)
		
	app.Listen(":5000")
}