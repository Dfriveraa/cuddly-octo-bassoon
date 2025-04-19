package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"tiny-url/internal/adapters/handlers"
	"tiny-url/internal/adapters/repository"
	"tiny-url/internal/domain/model"
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
	testDB *gorm.DB
)

// setupTestWithTransaction prepara un entorno aislado para cada test con su propia transacción
func setupTestWithTransaction(t *testing.T) (*gorm.DB, *gin.Engine, string, func()) {
	// Iniciar una transacción para aislar este test
	tx := testDB.Begin()
	require.NoError(t, tx.Error)

	// Inicializar los repositorios dentro de la transacción
	urlRepo := repository.NewURLRepository(tx)
	userRepo := repository.NewUserRepository(tx)

	// Inicializar los servicios
	urlService := service.NewURLService(urlRepo)
	authService := service.NewAuthService(userRepo)

	// Generar datos únicos para el test
	timestamp := time.Now().UnixNano()
	testUsername := fmt.Sprintf("testuser-%d", timestamp)
	testEmail := fmt.Sprintf("test-%d@example.com", timestamp)
	testPassword := "password123"

	// Crear usuario de prueba para autenticación
	testUser := &model.User{
		Username: testUsername,
		Email:    testEmail,
		Password: testPassword,
	}

	err := userRepo.CreateUser(testUser)
	require.NoError(t, err)

	// Obtener un token para las pruebas
	testToken, err := authService.Login(testUsername, testPassword)
	require.NoError(t, err)

	// Configurar el router para las pruebas
	r := gin.Default()

	// Configurar middleware de autenticación
	authMiddleware := func(c *gin.Context) {
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

	// Crear manejadores
	urlHandler := handlers.NewURLHandler(urlService)
	authHandler := handlers.NewAuthHandler(authService)

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

	// Función de cleanup que hace rollback de la transacción
	cleanup := func() {
		tx.Rollback()
	}

	return tx, r, testToken, cleanup
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
		log.Fatalf("Failed to start postgres container: %v", err)
	}

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("Failed to terminate container: %v", err)
		}
	}()

	// Obtener la información de conexión
	host, err := pgContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("Failed to get container port: %v", err)
	}

	// Crear cadena de conexión DSN
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	// Configurar la conexión a la base de datos
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrar los modelos
	if err := testDB.AutoMigrate(&model.URL{}, &model.User{}); err != nil {
		log.Fatalf("Failed to migrate models: %v", err)
	}

	// Ejecutar las pruebas
	exitCode := m.Run()

	// Salir con el código devuelto por las pruebas
	os.Exit(exitCode)
}

// Pruebas de integración para los endpoints de autenticación
func TestAuthHandler_Register(t *testing.T) {
	// Arrange
	_, router, _, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Crear usuario único para esta prueba
	timestamp := time.Now().UnixNano()
	username := fmt.Sprintf("newuser-%d", timestamp)
	email := fmt.Sprintf("new-%d@example.com", timestamp)

	// Crear solicitud
	registerData := map[string]string{
		"username": username,
		"email":    email,
		"password": "password123",
	}
	body, _ := json.Marshal(registerData)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Crear respuesta mock
	w := httptest.NewRecorder()

	// Act - Procesar la solicitud
	router.ServeHTTP(w, req)

	// Assert - Verificar respuesta
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Comprobar que tenemos token y datos de usuario
	assert.NotEmpty(t, response["token"])
	assert.NotNil(t, response["user"])
}

func TestAuthHandler_Login(t *testing.T) {
	// Arrange
	_, router, _, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Crear usuario único para esta prueba
	timestamp := time.Now().UnixNano()
	username := fmt.Sprintf("loginuser-%d", timestamp)
	email := fmt.Sprintf("login-%d@example.com", timestamp)
	password := "password123"

	// Crear el usuario directamente en la base de datos
	userRepo := repository.NewUserRepository(testDB)
	user := &model.User{
		Username: username,
		Email:    email,
		Password: password,
	}
	err := userRepo.CreateUser(user)
	require.NoError(t, err)

	// Crear solicitud
	loginData := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginData)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Crear respuesta mock
	w := httptest.NewRecorder()

	// Act - Procesar la solicitud
	router.ServeHTTP(w, req)

	// Assert - Verificar respuesta
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Comprobar que tenemos token y datos de usuario
	assert.NotEmpty(t, response["token"])
	assert.NotNil(t, response["user"])
}

func TestAuthHandler_GetUserProfile(t *testing.T) {
	// Arrange
	_, router, token, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Crear solicitud
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Crear respuesta mock
	w := httptest.NewRecorder()

	// Act - Procesar la solicitud
	router.ServeHTTP(w, req)

	// Assert - Verificar respuesta
	assert.Equal(t, http.StatusOK, w.Code)

	var user map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &user)
	assert.NoError(t, err)

	// Comprobar que los datos del usuario existen
	assert.NotEmpty(t, user["username"])
	assert.NotEmpty(t, user["email"])
}

// Pruebas de integración para los endpoints de URL
func TestURLHandler_ShortenAndGetURL(t *testing.T) {
	// Arrange
	_, router, token, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Generar URL única para esta prueba
	timestamp := time.Now().UnixNano()
	testUrl := fmt.Sprintf("https://www.example.com/test-integration-%d", timestamp)

	// Test 1: Acortar URL
	urlData := map[string]string{
		"url": testUrl,
	}
	body, _ := json.Marshal(urlData)
	req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar respuesta
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Comprobar que se creó la URL correctamente
	assert.Equal(t, testUrl, response["original_url"])
	shortCode := response["short_code"].(string)
	assert.NotEmpty(t, shortCode)

	// Test 2: Obtener información de la URL
	req = httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verificar respuesta
	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Comprobar la información de la URL
	assert.Equal(t, testUrl, response["original_url"])
	assert.Equal(t, shortCode, response["short_code"])
	assert.Equal(t, float64(0), response["visits"]) // Aún no se ha visitado
}

func TestURLHandler_RedirectURL(t *testing.T) {
	// Arrange
	_, router, token, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Primero crear una URL para redireccionar
	timestamp := time.Now().UnixNano()
	testUrl := fmt.Sprintf("https://www.example.com/redirect-test-%d", timestamp)

	urlData := map[string]string{
		"url": testUrl,
	}
	body, _ := json.Marshal(urlData)
	req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	shortCode := response["short_code"].(string)
	require.NotEmpty(t, shortCode)

	// Ahora probar la redirección
	req = httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
	w = httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, testUrl, w.Header().Get("Location"))
}

func TestURLHandler_ListURLs(t *testing.T) {
	// Arrange
	_, router, token, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Crear varias URLs para este test
	for i := 0; i < 3; i++ {
		testUrl := fmt.Sprintf("https://www.example.com/list-test-%d-%d", i, time.Now().UnixNano())
		urlData := map[string]string{
			"url": testUrl,
		}
		body, _ := json.Marshal(urlData)
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	// Act - Obtener la lista de URLs
	req := httptest.NewRequest(http.MethodGet, "/api/urls?limit=10&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Comprobar que recibimos la lista de URLs
	urls := response["urls"].([]interface{})
	assert.NotEmpty(t, urls)
	assert.Len(t, urls, 3) // Hemos creado 3 URLs en este test
}

func TestURLHandler_DeleteURL(t *testing.T) {
	// Arrange
	_, router, token, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	// Crear una URL para eliminar
	testUrl := fmt.Sprintf("https://www.example.com/delete-test-%d", time.Now().UnixNano())
	urlData := map[string]string{
		"url": testUrl,
	}
	body, _ := json.Marshal(urlData)
	req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	shortCode := createResponse["short_code"].(string)
	require.NotEmpty(t, shortCode)

	// Act - Eliminar la URL
	req = httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortCode, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	// Verificar que la URL ya no existe
	req = httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Pruebas para verificar la seguridad y autenticación de endpoints protegidos
func TestSecurityAndAuth(t *testing.T) {
	// Arrange
	_, router, _, cleanup := setupTestWithTransaction(t)
	defer cleanup()

	t.Run("ProtectedEndpoint_NoToken", func(t *testing.T) {
		// Intentar acceder a un endpoint protegido sin token
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verificar que se deniega el acceso
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ProtectedEndpoint_InvalidToken", func(t *testing.T) {
		// Intentar acceder con un token inválido
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verificar que se deniega el acceso
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("PublicEndpoint_NoAuth", func(t *testing.T) {
		// Las rutas públicas no deben requerir autenticación
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verificar acceso permitido
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
