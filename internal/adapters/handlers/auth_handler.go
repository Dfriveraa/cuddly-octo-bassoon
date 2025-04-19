package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
)

// AuthHandler maneja las peticiones HTTP relacionadas con la autenticación
type AuthHandler struct {
	authService ports.AuthService
}

// NewAuthHandler crea una nueva instancia del manejador de autenticación
func NewAuthHandler(authService ports.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// UserCredentials representa la solicitud de credenciales para login
type UserCredentials struct {
	Username string `json:"username" binding:"required" example:"usuario123"`
	Password string `json:"password" binding:"required" example:"contraseña123"`
}

// RegisterRequest representa la solicitud para registrar un nuevo usuario
type RegisterRequest struct {
	Username string `json:"username" binding:"required" example:"usuario123"`
	Email    string `json:"email" binding:"required,email" example:"usuario@ejemplo.com"`
	Password string `json:"password" binding:"required,min=6" example:"contraseña123"`
}

// AuthResponse representa la respuesta de autenticación con token JWT
type AuthResponse struct {
	User  interface{} `json:"user"`
	Token string      `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// handleAuthError centraliza el manejo de errores comunes en los handlers de autenticación
func (h *AuthHandler) handleAuthError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	// Errores específicos de autenticación
	if errors.Is(err, errors.ErrUserAlreadyExists) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El usuario o email ya existe",
		})
		return true
	}

	if errors.Is(err, errors.ErrInvalidCredentials) || errors.Is(err, errors.ErrUserNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Credenciales inválidas",
		})
		return true
	}

	// Error genérico del servidor
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Error del servidor",
	})
	return true
}

// createAuthResponse genera una respuesta de autenticación estandarizada
func (h *AuthHandler) createAuthResponse(c *gin.Context, user *model.User, token string, statusCode int) {
	// Ocultar la contraseña en la respuesta
	user.Password = ""

	c.JSON(statusCode, gin.H{
		"user":  user,
		"token": token,
	})
}

// Register godoc
// @Summary Registrar un nuevo usuario
// @Description Crea un nuevo usuario en el sistema y devuelve un token de autenticación
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Datos de registro del usuario"
// @Success 201 {object} AuthResponse "Usuario creado correctamente"
// @Failure 400 {object} map[string]string "Error en la solicitud"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos de registro inválidos",
		})
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), request.Username, request.Email, request.Password)
	if h.handleAuthError(c, err) {
		return
	}

	h.createAuthResponse(c, user, token, http.StatusCreated)
}

// Login godoc
// @Summary Iniciar sesión
// @Description Autentica a un usuario y devuelve un token JWT
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UserCredentials true "Credenciales de usuario"
// @Success 200 {object} AuthResponse "Inicio de sesión exitoso"
// @Failure 400 {object} map[string]string "Credenciales inválidas"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var creds UserCredentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Credenciales inválidas",
		})
		return
	}

	token, err := h.authService.Login(creds.Username, creds.Password)
	if h.handleAuthError(c, err) {
		return
	}

	// Obtener los datos del usuario a partir del token
	userID, err := h.authService.ValidateToken(token)
	if h.handleAuthError(c, err) {
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if h.handleAuthError(c, err) {
		return
	}

	h.createAuthResponse(c, user, token, http.StatusOK)
}

// GetUserProfile godoc
// @Summary Obtener perfil de usuario
// @Description Obtiene el perfil del usuario autenticado
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User "Perfil del usuario"
// @Failure 401 {object} map[string]string "No autenticado"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /api/profile [get]
func (h *AuthHandler) GetUserProfile(c *gin.Context) {
	// Obtener el ID del usuario del contexto (colocado por el middleware de autenticación)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No autenticado",
		})
		return
	}

	// Obtener información del usuario
	user, err := h.authService.GetUser(c.Request.Context(), userID.(uint))
	if h.handleAuthError(c, err) {
		return
	}

	// Ocultar la contraseña en la respuesta
	user.Password = ""

	c.JSON(http.StatusOK, user)
}
