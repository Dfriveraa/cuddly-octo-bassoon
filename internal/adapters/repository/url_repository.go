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
	db *gorm.DB
}

// NewURLRepository crea una nueva instancia del repositorio de URL
func NewURLRepository(db *gorm.DB) ports.URLRepository {
	return &URLRepository{
		db: db,
	}
}

// Create guarda una nueva URL en la base de datos
func (r *URLRepository) Create(ctx context.Context, url *model.URL) error {
	result := r.db.Create(url)
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al crear URL")
	}
	return nil
}

// GetByShortCode busca una URL por su código corto
func (r *URLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	var url model.URL
	result := r.db.Where("short_code = ?", shortCode).First(&url)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrURLNotFound
		}
		return nil, errors.Wrap(result.Error, "error al buscar URL por código corto")
	}
	return &url, nil
}

// GetByOriginalURL busca una URL por su URL original
func (r *URLRepository) GetByOriginalURL(ctx context.Context, originalURL string) (*model.URL, error) {
	var url model.URL
	result := r.db.Where("original_url = ?", originalURL).First(&url)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // No es un error, simplemente no existe
		}
		return nil, errors.Wrap(result.Error, "error al buscar URL por URL original")
	}
	return &url, nil
}

// IncrementVisits incrementa el contador de visitas de una URL
func (r *URLRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	result := r.db.Model(&model.URL{}).Where("short_code = ?", shortCode).
		UpdateColumn("visits", gorm.Expr("visits + ?", 1))
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al incrementar visitas")
	}
	if result.RowsAffected == 0 {
		return errors.ErrURLNotFound
	}
	return nil
}

// List obtiene todas las URLs con paginación
func (r *URLRepository) List(ctx context.Context, limit, offset int) ([]*model.URL, error) {
	var urls []*model.URL
	result := r.db.Limit(limit).Offset(offset).Find(&urls)
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "error al listar URLs")
	}
	return urls, nil
}

// Delete elimina una URL por su código corto
func (r *URLRepository) Delete(ctx context.Context, shortCode string) error {
	result := r.db.Where("short_code = ?", shortCode).Delete(&model.URL{})
	if result.Error != nil {
		return errors.Wrap(result.Error, "error al eliminar URL")
	}
	if result.RowsAffected == 0 {
		return errors.ErrURLNotFound
	}
	return nil
}
