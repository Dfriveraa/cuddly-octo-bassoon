package service

import (
	"context"
	"testing"

	domainErrors "tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestShortenURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	originalURL := "https://www.example.com/test"
	ctx := context.Background()

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByOriginalURL(ctx, originalURL).Return(nil, nil)
	mockRepo.EXPECT().Create(ctx, mock.AnythingOfType("*model.URL")).Return(nil)

	// Act
	url, err := service.ShortenURL(ctx, originalURL)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, originalURL, url.OriginalURL)
	assert.NotEmpty(t, url.ShortCode)
	assert.Equal(t, 0, url.Visits)
}

func TestShortenURL_EmptyURL(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	originalURL := ""
	ctx := context.Background()

	// Act
	url, err := service.ShortenURL(ctx, originalURL)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrInvalidURL))
	assert.Nil(t, url)
}

func TestShortenURL_ExistingURL(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	originalURL := "https://www.example.com/test"
	existingShortCode := "abc123"
	ctx := context.Background()

	existingURL := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   existingShortCode,
		Visits:      5,
	}

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByOriginalURL(ctx, originalURL).Return(existingURL, nil)

	// Act
	url, err := service.ShortenURL(ctx, originalURL)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, existingURL, url)
}

func TestGetURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "abc123"
	ctx := context.Background()

	expectedURL := &model.URL{
		OriginalURL: "https://www.example.com/test",
		ShortCode:   shortCode,
		Visits:      5,
	}

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByShortCode(ctx, shortCode).Return(expectedURL, nil)

	// Act
	url, err := service.GetURL(ctx, shortCode)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
}

func TestGetURL_NotFound(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "nonexistent"
	ctx := context.Background()

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByShortCode(ctx, shortCode).Return(nil, domainErrors.ErrURLNotFound)

	// Act
	url, err := service.GetURL(ctx, shortCode)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrURLNotFound))
	assert.Nil(t, url)
}

func TestRedirectURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "abc123"
	originalURL := "https://www.example.com/test"
	ctx := context.Background()

	// URL antes de incrementar visitas
	urlBeforeRedirect := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      5,
	}

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByShortCode(ctx, shortCode).Return(urlBeforeRedirect, nil)
	mockRepo.EXPECT().IncrementVisits(ctx, shortCode).Return(nil)

	// Act
	redirectURL, err := service.RedirectURL(ctx, shortCode)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, originalURL, redirectURL)
}

func TestRedirectURL_NotFound(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "nonexistent"
	ctx := context.Background()

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().GetByShortCode(ctx, shortCode).Return(nil, domainErrors.ErrURLNotFound)

	// Act
	redirectURL, err := service.RedirectURL(ctx, shortCode)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrURLNotFound))
	assert.Empty(t, redirectURL)
}

func TestListURLs_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	limit := 10
	offset := 0
	ctx := context.Background()

	expectedURLs := []*model.URL{
		{
			OriginalURL: "https://www.example.com/test1",
			ShortCode:   "abc123",
			Visits:      5,
		},
		{
			OriginalURL: "https://www.example.com/test2",
			ShortCode:   "def456",
			Visits:      3,
		},
	}

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().List(ctx, limit, offset).Return(expectedURLs, nil)

	// Act
	urls, err := service.ListURLs(ctx, limit, offset)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedURLs, urls)
	assert.Len(t, urls, 2)
}

func TestDeleteURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "abc123"
	ctx := context.Background()

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().Delete(ctx, shortCode).Return(nil)

	// Act
	err := service.DeleteURL(ctx, shortCode)

	// Assert
	assert.NoError(t, err)
}

func TestDeleteURL_NotFound(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	service := NewURLService(mockRepo)

	shortCode := "nonexistent"
	ctx := context.Background()

	// Configurar el comportamiento esperado del mock
	mockRepo.EXPECT().Delete(ctx, shortCode).Return(domainErrors.ErrURLNotFound)

	// Act
	err := service.DeleteURL(ctx, shortCode)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrURLNotFound))
}
