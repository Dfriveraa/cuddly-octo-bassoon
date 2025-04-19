package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	pg_container "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	testDB     *gorm.DB
	testDBConn *sql.DB
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Configurar el contenedor de PostgreSQL
	pgContainer, err := pg_container.RunContainer(ctx,
		testcontainers.WithImage("postgres:14-alpine"),
		pg_container.WithDatabase("testdb"),
		pg_container.WithUsername("testuser"),
		pg_container.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to start postgres container: %v", err)
	}

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("Failed to terminate container: %v", err)
		}
	}()

	// Obtener la cadena de conexión
	host, err := pgContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("Failed to get container port: %v", err)
	}

	dsn := "host=" + host + " port=" + port.Port() + " user=testuser password=testpass dbname=testdb sslmode=disable"

	// Inicializar la conexión de GORM
	var gormErr error
	testDB, gormErr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if gormErr != nil {
		log.Fatalf("Failed to connect to database: %v", gormErr)
	}

	// Migrar los modelos
	if err := testDB.AutoMigrate(&model.URL{}, &model.User{}); err != nil {
		log.Fatalf("Failed to migrate models: %v", err)
	}

	// Ejecutar los tests
	exitCode := m.Run()

	// Limpiar y salir
	testDBConn, err = testDB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	if err := testDBConn.Close(); err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}

	os.Exit(exitCode)
}

// setupTest crea una nueva transacción para aislar cada test
func setupTest(t *testing.T) (*gorm.DB, context.Context, func()) {
	// Iniciar una transacción para aislar este test
	tx := testDB.Begin()
	require.NoError(t, tx.Error)

	ctx := context.Background()

	// Función de cleanup que hace rollback de la transacción
	cleanup := func() {
		tx.Rollback()
	}

	return tx, ctx, cleanup
}

// generateUniqueData genera datos únicos para cada test
func generateUniqueData(testName string, index int) (string, string) {
	timestamp := time.Now().UnixNano()
	shortCode := fmt.Sprintf("t%s%d", testName[:1], timestamp%1000000000)[:10]
	originalURL := fmt.Sprintf("https://www.%s-%d.com", testName, index)
	return shortCode, originalURL
}

func TestURLRepository_Create_GetByShortCode(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)
	shortCode, originalURL := generateUniqueData("create-get", 1)

	url := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Act
	err := repo.Create(ctx, url)
	assert.NoError(t, err)

	// Retrieve the URL by short code
	retrievedURL, err := repo.GetByShortCode(ctx, url.ShortCode)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, retrievedURL)
	assert.Equal(t, url.OriginalURL, retrievedURL.OriginalURL)
	assert.Equal(t, url.ShortCode, retrievedURL.ShortCode)
	assert.Equal(t, url.Visits, retrievedURL.Visits)
}

func TestURLRepository_GetByOriginalURL(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)
	shortCode, originalURL := generateUniqueData("get-original", 1)

	url := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(ctx, url)
	require.NoError(t, err)

	// Act
	retrievedURL, err := repo.GetByOriginalURL(ctx, originalURL)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, retrievedURL)
	assert.Equal(t, url.OriginalURL, retrievedURL.OriginalURL)
	assert.Equal(t, url.ShortCode, retrievedURL.ShortCode)
}

func TestURLRepository_IncrementVisits(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)
	shortCode, originalURL := generateUniqueData("increment", 1)

	url := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(ctx, url)
	require.NoError(t, err)

	// Act
	err = repo.IncrementVisits(ctx, url.ShortCode)

	// Assert
	assert.NoError(t, err)

	// Verify the increment
	updatedURL, err := repo.GetByShortCode(ctx, url.ShortCode)
	assert.NoError(t, err)
	assert.Equal(t, url.Visits+1, updatedURL.Visits)
}

func TestURLRepository_List(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)

	// Create multiple URLs
	urls := []*model.URL{}
	for i := 0; i < 3; i++ {
		shortCode, originalURL := generateUniqueData("list", i)
		url := &model.URL{
			OriginalURL: originalURL,
			ShortCode:   shortCode,
			Visits:      i + 1,
			CreatedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
			UpdatedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
		}
		err := repo.Create(ctx, url)
		require.NoError(t, err)
		urls = append(urls, url)
	}

	// Act
	limit := 2
	offset := 0
	retrievedURLs, err := repo.List(ctx, limit, offset)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, retrievedURLs, limit)

	// El orden es descendente por fecha de creación
	assert.Equal(t, urls[0].ShortCode, retrievedURLs[0].ShortCode)
	assert.Equal(t, urls[1].ShortCode, retrievedURLs[1].ShortCode)

	// Prueba paginación
	offset = 2
	retrievedURLs, err = repo.List(ctx, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, retrievedURLs, 1)
	assert.Equal(t, urls[2].ShortCode, retrievedURLs[0].ShortCode)
}

func TestURLRepository_Delete(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)
	shortCode, originalURL := generateUniqueData("delete", 1)

	url := &model.URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Visits:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(ctx, url)
	require.NoError(t, err)

	// Act
	err = repo.Delete(ctx, url.ShortCode)

	// Assert
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByShortCode(ctx, url.ShortCode)
	assert.Error(t, err)
	assert.Equal(t, errors.ErrURLNotFound, err)
}

func TestURLRepository_GetByShortCode_NotFound(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)

	// Act
	_, err := repo.GetByShortCode(ctx, "nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrURLNotFound, err)
}

func TestURLRepository_Delete_NotFound(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewURLRepository(tx)

	// Act
	err := repo.Delete(ctx, "nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrURLNotFound, err)
}
