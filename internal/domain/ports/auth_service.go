package ports

import (
	"context"
	"tiny-url/internal/domain/model"
)

// AuthService define las operaciones para el servicio de autenticación
type AuthService interface {
	// Register registra un nuevo usuario
	Register(ctx context.Context, username, email, password string) (*model.User, string, error)

	// Login autentica a un usuario
	Login(username, password string) (string, error)

	// ValidateToken valida un token JWT y devuelve el ID del usuario
	ValidateToken(token string) (uint, error)
	GenerateToken(id uint) (string, error)
	// GetUser obtiene un usuario por su ID
	GetUser(ctx context.Context, id uint) (*model.User, error)
}
