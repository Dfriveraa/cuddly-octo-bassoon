package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/ports"
)

// URLHandler maneja las peticiones HTTP relacionadas con el acortador de URLs
type URLHandler struct {
	urlService ports.URLService
}

// NewURLHandler crea una nueva instancia del manejador de URLs
func NewURLHandler(urlService ports.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

// ShortenURLRequest representa la solicitud para acortar una URL
type ShortenURLRequest struct {
	URL string `json:"url" binding:"required,url" example:"https://www.ejemplo.com/pagina-con-url-muy-larga"`
}

// URLResponse representa la respuesta con la información de una URL acortada
type URLResponse struct {
	OriginalURL string `json:"original_url" example:"https://www.ejemplo.com/pagina-con-url-muy-larga"`
	ShortCode   string `json:"short_code" example:"abc123"`
	ShortURL    string `json:"short_url" example:"http://localhost:8080/abc123"`
	Visits      int    `json:"visits" example:"5"`
}

// ShortenURL godoc
// @Summary Acortar una URL
// @Description Crea una versión acortada de una URL proporcionada
// @Tags urls
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ShortenURLRequest true "URL a acortar"
// @Success 201 {object} URLResponse "URL acortada exitosamente"
// @Failure 400 {object} map[string]string "URL inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /api/urls [post]
func (h *URLHandler) ShortenURL(c *gin.Context) {
	var request ShortenURLRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL inválida",
		})
		return
	}

	url, err := h.urlService.ShortenURL(c.Request.Context(), request.URL)
	if err != nil {
		if errors.Is(err, errors.ErrInvalidURL) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "URL inválida",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al acortar la URL",
		})
		return
	}

	// Construir la URL acortada completa
	baseURL := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	shortURL := scheme + "://" + baseURL + "/" + url.ShortCode

	c.JSON(http.StatusCreated, gin.H{
		"original_url": url.OriginalURL,
		"short_code":   url.ShortCode,
		"short_url":    shortURL,
		"visits":       url.Visits,
	})
}

// RedirectURL godoc
// @Summary Redirigir a la URL original
// @Description Redirige al usuario a la URL original correspondiente al código corto
// @Tags redirection
// @Produce json
// @Param shortCode path string true "Código corto de la URL"
// @Success 301 "Redirección a la URL original"
// @Failure 404 {object} map[string]string "URL no encontrada"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /{shortCode} [get]
func (h *URLHandler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	originalURL, err := h.urlService.RedirectURL(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, errors.ErrURLNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "URL no encontrada",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al redirigir",
		})
		return
	}

	c.Redirect(http.StatusMovedPermanently, originalURL)
}

// GetURLInfo godoc
// @Summary Obtener información de una URL
// @Description Obtiene información detallada sobre una URL acortada
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param shortCode path string true "Código corto de la URL"
// @Success 200 {object} URLResponse "Información de la URL"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 404 {object} map[string]string "URL no encontrada"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /api/urls/{shortCode} [get]
func (h *URLHandler) GetURLInfo(c *gin.Context) {
	shortCode := c.Param("shortCode")
	url, err := h.urlService.GetURL(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, errors.ErrURLNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "URL no encontrada",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener información de la URL",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original_url": url.OriginalURL,
		"short_code":   url.ShortCode,
		"visits":       url.Visits,
		"created_at":   url.CreatedAt,
	})
}

// ListURLs godoc
// @Summary Listar todas las URLs
// @Description Obtiene una lista paginada de todas las URLs acortadas
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Límite de resultados por página (default: 10)"
// @Param offset query int false "Desplazamiento para paginación (default: 0)"
// @Success 200 {object} map[string]interface{} "Lista de URLs"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /api/urls [get]
func (h *URLHandler) ListURLs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	urls, err := h.urlService.ListURLs(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al listar URLs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"urls":   urls,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteURL godoc
// @Summary Eliminar una URL
// @Description Elimina una URL acortada por su código corto
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param shortCode path string true "Código corto de la URL"
// @Success 200 {object} map[string]string "URL eliminada correctamente"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 404 {object} map[string]string "URL no encontrada"
// @Failure 500 {object} map[string]string "Error del servidor"
// @Router /api/urls/{shortCode} [delete]
func (h *URLHandler) DeleteURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	err := h.urlService.DeleteURL(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, errors.ErrURLNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "URL no encontrada",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al eliminar la URL",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "URL eliminada correctamente",
	})
}
