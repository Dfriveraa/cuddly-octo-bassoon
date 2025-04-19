package repository

import (
	"context"
	"database/sql"
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

func TestURLRepository_Integration(t *testing.T) {
	// Limpiar la tabla antes de cada prueba
	testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")

	ctx := context.Background()
	repo := NewURLRepository(testDB)

	t.Run("Create and GetByShortCode", func(t *testing.T) {
		// Arrange
		url := &model.URL{
			OriginalURL: "https://www.example.com",
			ShortCode:   "test123",
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
	})

	t.Run("GetByOriginalURL", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")
		originalURL := "https://www.example2.com"
		url := &model.URL{
			OriginalURL: originalURL,
			ShortCode:   "test456",
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
	})

	t.Run("IncrementVisits", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")
		url := &model.URL{
			OriginalURL: "https://www.example3.com",
			ShortCode:   "test789",
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
	})

	t.Run("List", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")

		// Create multiple URLs
		urls := []*model.URL{
			{
				OriginalURL: "https://www.example1.com",
				ShortCode:   "list1",
				Visits:      1,
				CreatedAt:   time.Now().Add(-2 * time.Hour),
				UpdatedAt:   time.Now().Add(-2 * time.Hour),
			},
			{
				OriginalURL: "https://www.example2.com",
				ShortCode:   "list2",
				Visits:      2,
				CreatedAt:   time.Now().Add(-1 * time.Hour),
				UpdatedAt:   time.Now().Add(-1 * time.Hour),
			},
			{
				OriginalURL: "https://www.example3.com",
				ShortCode:   "list3",
				Visits:      3,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		for _, u := range urls {
			err := repo.Create(ctx, u)
			require.NoError(t, err)
		}

		// Act
		limit := 2
		offset := 0
		retrievedURLs, err := repo.List(ctx, limit, offset)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, retrievedURLs, limit)

		// El orden es descendente por fecha de creación
		assert.Equal(t, "list3", retrievedURLs[0].ShortCode)
		assert.Equal(t, "list2", retrievedURLs[1].ShortCode)

		// Prueba paginación
		offset = 2
		retrievedURLs, err = repo.List(ctx, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, retrievedURLs, 1)
		assert.Equal(t, "list1", retrievedURLs[0].ShortCode)
	})

	t.Run("Delete", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")
		url := &model.URL{
			OriginalURL: "https://www.example-delete.com",
			ShortCode:   "deltest",
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
	})

	t.Run("GetByShortCode_NotFound", func(t *testing.T) {
		// Act
		_, err := repo.GetByShortCode(ctx, "nonexistent")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, errors.ErrURLNotFound, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		// Act
		err := repo.Delete(ctx, "nonexistent")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, errors.ErrURLNotFound, err)
	})
}
