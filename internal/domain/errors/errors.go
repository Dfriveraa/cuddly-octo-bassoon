// Package errors define los errores comunes utilizados en toda la aplicación.
package errors

import (
	"errors"
)

// Errores comunes de la aplicación
var (
	// Errores del servicio de URL
	ErrURLNotFound    = errors.New("url not found")
	ErrInvalidURL     = errors.New("invalid url")
	ErrGeneratingCode = errors.New("error generating short code")

	// Errores del servicio de autenticación
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")

	// Errores de base de datos
	ErrDatabaseConnection = errors.New("database connection error")
	ErrRecordNotFound     = errors.New("record not found")
	ErrDuplicateKey       = errors.New("duplicate key error")

	// Errores generales
	ErrInternalServer = errors.New("internal server error")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
)

// New crea un nuevo error con el mensaje especificado.
func New(text string) error {
	return errors.New(text)
}

// Wrap envuelve un error con un mensaje descriptivo.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return errors.New(message + ": " + err.Error())
}

// Is reporta si algún error en la cadena de error coincide con el error objetivo.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As encuentra el primer error en la cadena de error que coincide con el tipo objetivo.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
