package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/ports"
)

// AuthMiddleware crea un middleware para proteger rutas que requieren autenticación
func AuthMiddleware(authService ports.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extraer token del encabezado "Authorization"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No autorizado: token no proporcionado"})
			c.Abort()
			return
		}

		// El formato del encabezado debe ser "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de token inválido"})
			c.Abort()
			return
		}

		// Obtener el token
		tokenString := parts[1]

		// Validar el token y obtener el ID del usuario
		userID, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, errors.ErrInvalidToken) || errors.Is(err, errors.ErrExpiredToken) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Error al validar token"})
			}
			c.Abort()
			return
		}

		// Almacenar el ID del usuario en el contexto para usarlo en los controladores
		c.Set("userID", userID)

		// Continuar con la solicitud
		c.Next()
	}
}
