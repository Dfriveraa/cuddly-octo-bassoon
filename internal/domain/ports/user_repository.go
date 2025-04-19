package ports

import (
	"context"
	"tiny-url/internal/domain/model"
)

// UserRepository define las operaciones para el repositorio de usuarios
type UserRepository interface {
	// CreateUser crea un nuevo usuario en la base de datos
	CreateUser(user *model.User) error

	// GetByID obtiene un usuario por su ID
	GetByID(ctx context.Context, id uint) (*model.User, error)

	// GetByUsername obtiene un usuario por su nombre de usuario
	GetByUsername(ctx context.Context, username string) (*model.User, error)

	// GetByEmail obtiene un usuario por su correo electrónico
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// UpdateUser actualiza la información de un usuario
	UpdateUser(user *model.User) error

	// DeleteUser elimina un usuario de la base de datos
	DeleteUser(id uint) error
}
