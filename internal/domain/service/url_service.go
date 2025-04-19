package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
)

const (
	// Caracteres permitidos para códigos cortos
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Longitud del código corto
	codeLength = 6
)

type urlService struct {
	repo ports.URLRepository
}

// NewURLService crea una nueva instancia del servicio de URL
func NewURLService(repo ports.URLRepository) ports.URLService {
	return &urlService{
		repo: repo,
	}
}

// ShortenURL implementa la lógica para acortar una URL
func (s *urlService) ShortenURL(ctx context.Context, originalURL string) (*model.URL, error) {
	// Validar que la URL no esté vacía
	if originalURL == "" {
		return nil, errors.ErrInvalidURL
	}

	// Verificar si la URL ya existe en la base de datos
	existingURL, err := s.repo.GetByOriginalURL(ctx, originalURL)
	if err == nil && existingURL != nil {
		return existingURL, nil
	}

	// Generar un código corto único
	shortCode, err := s.generateShortCode()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrGeneratingCode, err)
	}

	// Crear el objeto URL
	url := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      0,
	}

	// Guardar la URL en el repositorio
	if err := s.repo.Create(ctx, url); err != nil {
		return nil, err
	}

	return url, nil
}

// GetURL recupera una URL por su código corto
func (s *urlService) GetURL(ctx context.Context, shortCode string) (*model.URL, error) {
	url, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	if url == nil {
		return nil, errors.ErrURLNotFound
	}
	return url, nil
}

// RedirectURL recupera la URL original y aumenta el contador de visitas
func (s *urlService) RedirectURL(ctx context.Context, shortCode string) (string, error) {
	url, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if url == nil {
		return "", errors.ErrURLNotFound
	}

	// Incrementar el contador de visitas
	if err := s.repo.IncrementVisits(ctx, shortCode); err != nil {
		// Simplemente lo registramos pero no fallamos la redirección
		fmt.Printf("Error incrementando visitas: %v\n", err)
	}

	return url.OriginalURL, nil
}

// ListURLs recupera todas las URLs con paginación
func (s *urlService) ListURLs(ctx context.Context, limit, offset int) ([]*model.URL, error) {
	return s.repo.List(ctx, limit, offset)
}

// DeleteURL elimina una URL por su código corto
func (s *urlService) DeleteURL(ctx context.Context, shortCode string) error {
	return s.repo.Delete(ctx, shortCode)
}

// generateShortCode genera un código corto único para la URL
func (s *urlService) generateShortCode() (string, error) {
	code := make([]byte, codeLength)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < codeLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		code[i] = charset[randomIndex.Int64()]
	}

	return string(code), nil
}
