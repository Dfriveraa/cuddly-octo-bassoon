package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"tiny-url/internal/adapters/handlers"
	"tiny-url/internal/adapters/repository"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
	"tiny-url/internal/domain/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	pg_container "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	testRouter     http.Handler
	testDB         *gorm.DB
	testToken      string
	authMiddleware gin.HandlerFunc
)

// setupRouter crea un router de Gin con todas las rutas configuradas
func setupRouter(urlService ports.URLService, authService ports.AuthService) http.Handler {
	r := gin.Default()

	// Crear manejadores
	urlHandler := handlers.NewURLHandler(urlService)
	authHandler := handlers.NewAuthHandler(authService)

	// Configurar middleware de autenticación
	authMiddleware = func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No autorizado"})
			c.Abort()
			return
		}

		// Extraer el token del encabezado (formato "Bearer token")
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token no proporcionado"})
			c.Abort()
			return
		}

		// Validar el token
		userID, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		// Almacenar el ID del usuario en el contexto para usarlo luego
		c.Set("userID", userID)
		c.Next()
	}

	// Configurar rutas
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Rutas de autenticación (públicas)
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Rutas para el acortador de URLs
	api := r.Group("/api")
	{
		// Ruta de perfil de usuario (requiere autenticación)
		api.GET("/profile", authMiddleware, authHandler.GetUserProfile)

		// Rutas para URLs (requieren autenticación)
		urls := api.Group("/urls")
		urls.Use(authMiddleware) // Aplicar middleware de autenticación a todas las rutas de URLs
		{
			// Acortar URL
			urls.POST("", urlHandler.ShortenURL)

			// Listar todas las URLs acortadas
			urls.GET("", urlHandler.ListURLs)

			// Obtener información de una URL acortada
			urls.GET("/:shortCode", urlHandler.GetURLInfo)

			// Eliminar una URL acortada
			urls.DELETE("/:shortCode", urlHandler.DeleteURL)
		}
	}

	// Ruta para redireccionar usando el código corto (pública)
	r.GET("/:shortCode", urlHandler.RedirectURL)

	return r
}

func TestMain(m *testing.M) {
	// Configurar el entorno de prueba
	gin.SetMode(gin.TestMode)

	// Configurar el contenedor PostgreSQL
	ctx := context.Background()

	pgContainer, err := pg_container.Run(ctx,
		"postgres:14-alpine",
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
		fmt.Printf("Failed to start Postgres container: %v\n", err)
		os.Exit(1)
	}

	// Limpiar el contenedor al finalizar
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate container: %v\n", err)
		}
	}()

	// Obtener la información de conexión
	host, err := pgContainer.Host(ctx)
	if err != nil {
		fmt.Printf("Failed to get container host: %v\n", err)
		os.Exit(1)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Printf("Failed to get container port: %v\n", err)
		os.Exit(1)
	}

	// Crear cadena de conexión DSN
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	// Configurar la conexión a la base de datos
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Migrar los modelos
	if err := testDB.AutoMigrate(&model.URL{}, &model.User{}); err != nil {
		fmt.Printf("Failed to migrate models: %v\n", err)
		os.Exit(1)
	}

	// Inicializar los repositorios
	urlRepo := repository.NewURLRepository(testDB)
	userRepo := repository.NewUserRepository(testDB)

	// Inicializar los servicios
	urlService := service.NewURLService(urlRepo)
	authService := service.NewAuthService(userRepo)

	// Crear usuario de prueba para autenticación
	testUser := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123", // En un entorno real, esto sería hasheado
	}
	if err := userRepo.CreateUser(testUser); err != nil {
		fmt.Printf("Failed to create test user: %v\n", err)
		os.Exit(1)
	}

	// Obtener un token para las pruebas
	var tokenErr error
	testToken, tokenErr = authService.Login("testuser", "password123")
	if tokenErr != nil {
		fmt.Printf("Failed to generate test token: %v\n", tokenErr)
		os.Exit(1)
	}

	// Configurar el router para las pruebas
	testRouter = setupRouter(urlService, authService)

	// Ejecutar las pruebas
	exitCode := m.Run()

	// Salir con el código devuelto por las pruebas
	os.Exit(exitCode)
}

// Pruebas de integración para los endpoints de autenticación
func TestAuthHandler_Integration(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		// Limpiar datos previos
		testDB.Exec("DELETE FROM users WHERE username = ?", "newuser")

		// Crear solicitud
		registerData := map[string]string{
			"username": "newuser",
			"email":    "new@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(registerData)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Comprobar que tenemos token y datos de usuario
		assert.NotEmpty(t, response["token"])
		assert.NotNil(t, response["user"])
	})

	t.Run("Login", func(t *testing.T) {
		// Crear solicitud
		loginData := map[string]string{
			"username": "testuser",
			"password": "password123",
		}
		body, _ := json.Marshal(loginData)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Comprobar que tenemos token y datos de usuario
		assert.NotEmpty(t, response["token"])
		assert.NotNil(t, response["user"])
	})

	t.Run("GetUserProfile", func(t *testing.T) {
		// Crear solicitud
		req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusOK, w.Code)

		var user map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &user)
		assert.NoError(t, err)

		// Comprobar que los datos del usuario son correctos
		assert.Equal(t, "testuser", user["username"])
		assert.Equal(t, "test@example.com", user["email"])
	})
}

// Pruebas de integración para los endpoints de URL
func TestURLHandler_Integration(t *testing.T) {
	// Limpiar tabla de URLs para pruebas limpias
	testDB.Exec("TRUNCATE TABLE urls RESTART IDENTITY")

	var shortCode string

	t.Run("ShortenURL", func(t *testing.T) {
		// Crear solicitud
		urlData := map[string]string{
			"url": "https://www.example.com/test-integration",
		}
		body, _ := json.Marshal(urlData)
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Comprobar que se creó la URL correctamente
		assert.Equal(t, urlData["url"], response["original_url"])
		assert.NotEmpty(t, response["short_code"])

		// Guardar el short_code para pruebas posteriores
		shortCode = response["short_code"].(string)
	})

	t.Run("GetURLInfo", func(t *testing.T) {
		// Comprobar que tenemos un shortCode válido
		require.NotEmpty(t, shortCode)

		// Crear solicitud
		req := httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode, nil)
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Comprobar la información de la URL
		assert.Equal(t, "https://www.example.com/test-integration", response["original_url"])
		assert.Equal(t, shortCode, response["short_code"])
		assert.Equal(t, float64(0), response["visits"]) // Aún no se ha visitado
	})

	t.Run("RedirectURL", func(t *testing.T) {
		// Comprobar que tenemos un shortCode válido
		require.NotEmpty(t, shortCode)

		// Crear solicitud
		req := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar la redirección
		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "https://www.example.com/test-integration", w.Header().Get("Location"))
	})

	t.Run("ListURLs", func(t *testing.T) {
		// Crear solicitud
		req := httptest.NewRequest(http.MethodGet, "/api/urls?limit=10&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Comprobar que recibimos la lista de URLs
		urls := response["urls"].([]interface{})
		assert.NotEmpty(t, urls)
		assert.Len(t, urls, 1) // Solo hemos creado 1 URL en este test
	})

	t.Run("DeleteURL", func(t *testing.T) {
		// Crear una URL primero para asegurarnos de tener algo para eliminar
		// Crear solicitud para acortar URL
		urlData := map[string]string{
			"url": "https://www.example.com/url-to-delete",
		}
		body, _ := json.Marshal(urlData)
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock
		w := httptest.NewRecorder()

		// Procesar la solicitud para crear URL
		testRouter.ServeHTTP(w, req)

		// Verificar que la URL se creó correctamente
		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		assert.NoError(t, err)

		// Obtener el shortCode de la URL recién creada
		deleteShortCode := createResponse["short_code"].(string)
		assert.NotEmpty(t, deleteShortCode, "El shortCode no debe estar vacío")

		// Ahora intentar eliminar la URL
		req = httptest.NewRequest(http.MethodDelete, "/api/urls/"+deleteShortCode, nil)
		req.Header.Set("Authorization", "Bearer "+testToken)

		// Crear respuesta mock para delete
		w = httptest.NewRecorder()

		// Procesar la solicitud de eliminación
		testRouter.ServeHTTP(w, req)

		// Verificar respuesta
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verificar que existe un mensaje en la respuesta
		message, exists := response["message"]
		assert.True(t, exists, "La respuesta debe contener un campo 'message'")
		if exists {
			messageStr, ok := message.(string)
			assert.True(t, ok, "El campo 'message' debe ser una cadena de texto")
			if ok {
				assert.Contains(t, messageStr, "eliminada", "El mensaje debe indicar que la URL fue eliminada")
			}
		}

		// Verificar que la URL ya no existe
		req = httptest.NewRequest(http.MethodGet, "/api/urls/"+deleteShortCode, nil)
		req.Header.Set("Authorization", "Bearer "+testToken)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// Pruebas para verificar la seguridad y autenticación de endpoints protegidos
func TestSecurityAndAuth_Integration(t *testing.T) {
	t.Run("ProtectedEndpoint_NoToken", func(t *testing.T) {
		// Intentar acceder a un endpoint protegido sin token
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		// Verificar que se deniega el acceso
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ProtectedEndpoint_InvalidToken", func(t *testing.T) {
		// Intentar acceder con un token inválido
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		// Verificar que se deniega el acceso
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("PublicEndpoint_NoAuth", func(t *testing.T) {
		// Las rutas públicas no deben requerir autenticación
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		// Verificar acceso permitido
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
