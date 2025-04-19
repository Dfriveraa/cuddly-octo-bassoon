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
	db *gorm.DB
}

// NewUserRepository crea una nueva instancia del repositorio de usuario
func NewUserRepository(db *gorm.DB) ports.UserRepository {
	return &UserRepository{
		db: db,
	}
}

// CreateUser crea un nuevo usuario en la base de datos
func (r *UserRepository) CreateUser(user *model.User) error {
	result := r.db.Create(user)
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al crear usuario")
	}
	return nil
}

// GetByID busca un usuario por su ID
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	result := r.db.First(&user, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(result.Error, "error al buscar usuario por ID")
	}
	return &user, nil
}

// GetByUsername busca un usuario por su nombre de usuario
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	result := r.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(result.Error, "error al buscar usuario por nombre de usuario")
	}
	return &user, nil
}

// GetByEmail busca un usuario por su email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(result.Error, "error al buscar usuario por email")
	}
	return &user, nil
}

// UpdateUser actualiza un usuario existente
func (r *UserRepository) UpdateUser(user *model.User) error {
	result := r.db.Save(user)
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al actualizar usuario")
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}

// DeleteUser elimina un usuario por su ID
func (r *UserRepository) DeleteUser(id uint) error {
	result := r.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al eliminar usuario")
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound
	}
	return nil
}
