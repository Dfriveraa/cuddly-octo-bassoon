package repository

import (
	"context"

	"gorm.io/gorm"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
)

// URLRepository implementa ports.URLRepository
type URLRepository struct {
	BaseRepository
}

// NewURLRepository crea una nueva instancia del repositorio de URL
func NewURLRepository(db *gorm.DB) ports.URLRepository {
	return &URLRepository{
		BaseRepository: newBaseRepository(db),
	}
}

// Create guarda una nueva URL en la base de datos
func (r *URLRepository) Create(ctx context.Context, url *model.URL) error {
	err := r.create(url)
	return r.handleGormError(err, nil, "error al crear URL")
}

// GetByShortCode busca una URL por su código corto
func (r *URLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	var url model.URL
	err := r.findOne(&url, "short_code = ?", shortCode)
	if err := r.handleGormError(err, errors.ErrURLNotFound, "error al buscar URL por código corto"); err != nil {
		return nil, err
	}
	return &url, nil
}

// GetByOriginalURL busca una URL por su URL original
func (r *URLRepository) GetByOriginalURL(ctx context.Context, originalURL string) (*model.URL, error) {
	var url model.URL
	err := r.findOne(&url, "original_url = ?", originalURL)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No es un error, simplemente no existe
		}
		return nil, errors.Wrap(err, "error al buscar URL por URL original")
	}
	return &url, nil
}

// IncrementVisits incrementa el contador de visitas de una URL
func (r *URLRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	rowsAffected, err := r.updateColumn(&model.URL{}, "short_code = ?", "visits", gorm.Expr("visits + ?", 1), shortCode)
	if err != nil {
		return errors.Wrap(err, "error al incrementar visitas")
	}
	if rowsAffected == 0 {
		return errors.ErrURLNotFound
	}
	return nil
}

// List obtiene todas las URLs con paginación
func (r *URLRepository) List(ctx context.Context, limit, offset int) ([]*model.URL, error) {
	var urls []*model.URL
	err := r.findAll(&urls, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "error al listar URLs")
	}
	return urls, nil
}

// Delete elimina una URL por su código corto
func (r *URLRepository) Delete(ctx context.Context, shortCode string) error {
	rowsAffected, err := r.delete(&model.URL{}, "short_code = ?", shortCode)
	if err != nil {
		return errors.Wrap(err, "error al eliminar URL")
	}
	if rowsAffected == 0 {
		return errors.ErrURLNotFound
	}
	return nil
}
