package ports

import (
"context"

"tiny-url/internal/domain/model"
)

// URLService define las operaciones de negocio para el acortador de URLs
type URLService interface {
	// ShortenURL crea una URL acortada para una URL original
	ShortenURL(ctx context.Context, originalURL string) (*model.URL, error)
	
	// GetURL recupera la URL original a partir del código corto
	GetURL(ctx context.Context, shortCode string) (*model.URL, error)
	
	// RedirectURL recupera la URL original y actualiza el contador de visitas
	RedirectURL(ctx context.Context, shortCode string) (string, error)
	
	// ListURLs recupera todas las URLs con opciones de paginación
	ListURLs(ctx context.Context, limit, offset int) ([]*model.URL, error)
	
	// DeleteURL elimina una URL por su código corto
	DeleteURL(ctx context.Context, shortCode string) error
}
