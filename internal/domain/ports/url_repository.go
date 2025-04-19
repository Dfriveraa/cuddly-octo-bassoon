package ports

import (
"context"

"tiny-url/internal/domain/model"
)

// URLRepository define las operaciones que debe implementar cualquier repositorio para las URLs
type URLRepository interface {
	// Create guarda una nueva URL en el repositorio
	Create(ctx context.Context, url *model.URL) error
	
	// GetByShortCode recupera una URL por su código corto
	GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error)
	
	// GetByOriginalURL recupera una URL por su URL original
	GetByOriginalURL(ctx context.Context, originalURL string) (*model.URL, error)
	
	// IncrementVisits incrementa el contador de visitas para una URL
	IncrementVisits(ctx context.Context, shortCode string) error
	
	// List recupera todas las URLs con opciones de paginación
	List(ctx context.Context, limit, offset int) ([]*model.URL, error)
	
	// Delete elimina una URL por su código corto
	Delete(ctx context.Context, shortCode string) error
}
