package handler

import (
	"Shortener/internal/models"
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

var (
	ErrParseBody      = errors.New("не удалось разобрать тело запроса")
	ErrEmptyURL       = errors.New("url не может быть пустым")
	ErrEmptyShortCode = errors.New("short code не может быть пустым")
	ErrInvalidURL     = errors.New("некорректный url")
)

type URLService interface {
	CreateURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string, r *http.Request) (string, error)
	GetAnalytics(ctx context.Context, shortCode string) (*models.AnalyticsResponse, error)
}

type URLHandler struct {
	service URLService
}

func New(service URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

func (h *URLHandler) CreateURL(c *ginext.Context) {
	var req models.ShortenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrParseBody.Error(),
		})
		return
	}

	if req.LongURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrEmptyURL.Error(),
		})
		return
	}

	parsedURL, err := url.ParseRequestURI(req.LongURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrInvalidURL.Error(),
		})
		return
	}

	req.LongURL = parsedURL.String()

	response, err := h.service.CreateURL(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *URLHandler) GetOriginalURL(c *ginext.Context) {
	shortCode := c.Param("short_code")

	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrEmptyShortCode.Error(),
		})
		return
	}

	originalURL, err := h.service.GetOriginalURL(
		c.Request.Context(),
		shortCode,
		c.Request,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func (h *URLHandler) GetAnalytics(c *ginext.Context) {
	shortCode := c.Param("short_code")

	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrEmptyShortCode.Error(),
		})
		return
	}

	analyticsResponse, err := h.service.GetAnalytics(
		c.Request.Context(),
		shortCode,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analyticsResponse)
}
