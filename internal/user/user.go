package user

import (
	"context"
	"core-regulus-backend/internal/db"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type InAuthRequest struct {
	Name  string `json:"username" validate:"required,min=3,max=20"`
	Email string `json:"email" validate:"required,email"`
}

func validateUser(user InAuthRequest) error {
	validate := validator.New()
	return validate.Struct(user)
}

type ErrorResponse struct {
	Error       bool
	FailedField string
	Value       any
	Tag         string
}

type UserData struct {
	Name  string `json:"name"`
	Id    string `json:"id"`
	Email string `json:"email"`
}

var secretKey = []byte("SecretKey")

func postUserAuthHandler(c *fiber.Ctx) error {
	var authReq InAuthRequest

	if err := c.BodyParser(&authReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}
	if errs := validateUser(authReq); errs != nil {
		validationErrors := []ErrorResponse{}
		for _, err := range errs.(validator.ValidationErrors) {
			var elem ErrorResponse
			elem.FailedField = err.Field()
			elem.Value = err.Value()
			elem.Error = true
			elem.Tag = err.Tag()
			validationErrors = append(validationErrors, elem)
		}
		return c.Status(fiber.StatusBadRequest).JSON(validationErrors)
	}
	pool := db.Connect()
	ctx := context.Background()
	var user UserData
	err := pool.QueryRow(ctx, "select users.set_user($1)", authReq).Scan(&user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err})
	}

	claims := &jwt.MapClaims{
		"user":  user.Name,
		"email": user.Email,
		"id":    user.Id,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Cannot create jwt token",
		})
	}
	return c.Status(201).JSON(fiber.Map{"status": "OK", "token": tokenString})
}

func InitRoutes(app *fiber.App) {
	app.Post("/user/auth", postUserAuthHandler)
}
