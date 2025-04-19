package server

import (
	"net/http"

	_ "tiny-url/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"tiny-url/internal/adapters/handlers"
)

// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server cellear server.
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @securitydefinitions.oauth2.password	OPasswordAuth
// @tokenUrl								/auth/login
// @description							JWT Token created by username and password
func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Endpoint para la documentación Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Crear manejadores
	urlHandler := handlers.NewURLHandler(s.urlService)
	authHandler := handlers.NewAuthHandler(s.authService)

	// Ruta raíz para información general
	// @Summary Información general de la API
	// @Description Devuelve información básica sobre el servicio Tiny URL
	// @Tags info
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router / [get]
	r.GET("/", s.HelloWorldHandler)

	// Rutas de salud y estado
	// @Summary Estado del servicio
	// @Description Verifica el estado de salud del servicio y la conexión a la base de datos
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /health [get]
	r.GET("/health", s.healthHandler)

	// Rutas de autenticación (públicas)
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Middleware de autenticación para rutas protegidas
	authRequired := AuthMiddleware(s.authService)

	// Rutas para el acortador de URLs
	api := r.Group("/api")
	{
		// Ruta de perfil de usuario (requiere autenticación)
		api.GET("/profile", authRequired, authHandler.GetUserProfile)

		// Rutas para URLs (requieren autenticación)
		urls := api.Group("/urls")
		urls.Use(authRequired) // Aplicar middleware de autenticación a todas las rutas de URLs
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

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Tiny URL - Acortador de URLs"
	resp["version"] = "1.0.0"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}
