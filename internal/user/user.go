package user

import (
	"context"
	"core-regulus-backend/internal/db"
	"core-regulus-backend/internal/token"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type InAuthRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Description string `json:"description"`
	Agent string `json:"userAgent"`
	Id string `json:"id"`
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

func removePrefix(data string, prefix string) string {	
	if after, ok :=strings.CutPrefix(data, prefix); ok  {
		return after	
	}
	if after, ok :=strings.CutPrefix(data, strings.ToLower(prefix)); ok  {
		return after	
	}

	return ""	
}

func getBearerTokenString(c *fiber.Ctx) (string, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	val := removePrefix(authHeader, "Bearer ")
			
	if (val == "") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	return val, nil
}

func getBearerToken(c *fiber.Ctx) (*token.UserTokenData, error) {
	tokenString, err := getBearerTokenString(c)
	if (err != nil) {
		return nil, err
	}
	return token.ValidateJWT(tokenString)			
}

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

	tokenData, _ := getBearerToken(c)
	if (tokenData != nil) {
		authReq.Id = tokenData.Id
	} else {
		authReq.Id = ""
	}
	
	pool := db.Connect()
	ctx := context.Background()
	var user token.UserTokenData
	err := pool.QueryRow(ctx, "select users.set_user($1)", authReq).Scan(&user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err})
	}
		
	tokenString,err := token.GenerateJWT(user)	
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
