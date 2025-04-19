package repository

import (
	"context"

	"gorm.io/gorm"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
)

// UserRepository implementa ports.UserRepository
type UserRepository struct {
	BaseRepository
}

// NewUserRepository crea una nueva instancia del repositorio de usuario
func NewUserRepository(db *gorm.DB) ports.UserRepository {
	return &UserRepository{
		BaseRepository: newBaseRepository(db),
	}
}

// CreateUser crea un nuevo usuario en la base de datos
func (r *UserRepository) CreateUser(user *model.User) error {
	err := r.create(user)
	return r.handleGormError(err, nil, "error al crear usuario")
}

// GetByID busca un usuario por su ID
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.findById(&user, id)
	if err := r.handleGormError(err, errors.ErrUserNotFound, "error al buscar usuario por ID"); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername busca un usuario por su nombre de usuario
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.findOne(&user, "username = ?", username)
	if err := r.handleGormError(err, errors.ErrUserNotFound, "error al buscar usuario por nombre de usuario"); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail busca un usuario por su email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.findOne(&user, "email = ?", email)
	if err := r.handleGormError(err, errors.ErrUserNotFound, "error al buscar usuario por email"); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser actualiza un usuario existente
func (r *UserRepository) UpdateUser(user *model.User) error {
	rowsAffected, err := r.update(user)
	if err != nil {
		return errors.Wrap(err, "error al actualizar usuario")
	}
	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}

// DeleteUser elimina un usuario por su ID
func (r *UserRepository) DeleteUser(id uint) error {
	rowsAffected, err := r.deleteById(&model.User{}, id)
	if err != nil {
		return errors.Wrap(err, "error al eliminar usuario")
	}
	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}
