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
	ErrParseBody            = errors.New("ошибка парса тела запроса")
	ErrEmptyStringURl       = errors.New("пустая строка url")
	ErrEmptyStringShortCode = errors.New("пустая строка short code")
	ErrNotValidURL          = errors.New("невалидный url")
)

type URLService interface {
	CreateURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string, r *http.Request) (string, error)
	GetAnalytics(ctx context.Context, shortCode string) (*models.AnalyticsResponse, error)
}

type URlHandler struct {
	service URLService
}

func New(service URLService) URlHandler {
	return URlHandler{service: service}
}

func (h *URlHandler) CreateURL(c *ginext.Context) {
	var req models.ShortenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrParseBody.Error()})
		return
	}

	if req.LongURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrEmptyStringURl.Error()})
		return
	}

	validURL, err := url.ParseRequestURI(req.LongURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrNotValidURL.Error()})
		return
	}
	req.LongURL = validURL.String()

	response, err := h.service.CreateURL(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *URlHandler) GetOriginalURL(c *gin.Context) {
	shortCode := c.Param("short_code")

	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrEmptyStringShortCode.Error()})
		return
	}

	originalURL, err := h.service.GetOriginalURL(c.Request.Context(), shortCode, c.Request)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func (h *URlHandler) GetAnalytics(c *gin.Context) {
	shortCode := c.Param("short_code")

	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrEmptyStringShortCode.Error()})
		return
	}

	analiticsResponce, err := h.service.GetAnalytics(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analiticsResponce)
}
