package user

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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
	return c.Status(200).JSON(fiber.Map{"status": "OK"})

}

func InitRoutes(app *fiber.App) {
	app.Post("/user/auth", postUserAuthHandler)
}
