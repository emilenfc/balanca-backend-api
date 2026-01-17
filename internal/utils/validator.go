package utils

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

func init() {
	// Register custom validations
	validate.RegisterValidation("uuid", func(fl validator.FieldLevel) bool {
		field := fl.Field().String()
		_, err := uuid.Parse(field)
		return err == nil
	})
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
