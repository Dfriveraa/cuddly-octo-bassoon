package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"

	"tiny-url/internal/adapters/repository"
	"tiny-url/internal/database"
	"tiny-url/internal/domain/ports"
	"tiny-url/internal/domain/service"
)

type Server struct {
	port int

	db          database.Service
	gormDB      *database.GormService
	urlService  ports.URLService
	authService ports.AuthService
	userRepo    ports.UserRepository
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	// Inicializar la base de datos tradicional
	dbService := database.New()

	// Inicializar GORM para PostgreSQL
	gormService := database.NewGormService()

	// Inicializar el repositorio de URLs
	urlRepository := repository.NewURLRepository(gormService.GetDB())

	// Inicializar el repositorio de usuarios
	userRepository := repository.NewUserRepository(gormService.GetDB())

	// Inicializar los servicios
	urlService := service.NewURLService(urlRepository)
	authService := service.NewAuthService(userRepository)

	// Crear la instancia del servidor
	newServer := &Server{
		port:        port,
		db:          dbService,
		gormDB:      gormService,
		urlService:  urlService,
		authService: authService,
		userRepo:    userRepository,
	}

	// Configurar el servidor HTTP
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", newServer.port),
		Handler:      newServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

// NewServerWithDependencies crea una instancia del servidor con dependencias inyectadas
// Útil para pruebas de integración y entornos controlados
func NewServerWithDependencies(db *gorm.DB, urlService ports.URLService, authService ports.AuthService) *Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080 // Puerto por defecto para pruebas
	}

	// Usar una implementación ficticia para la base de datos tradicional en pruebas
	dbService := database.New()

	// Crear la instancia del servidor con las dependencias inyectadas
	return &Server{
		port:        port,
		db:          dbService,
		urlService:  urlService,
		authService: authService,
	}
}
