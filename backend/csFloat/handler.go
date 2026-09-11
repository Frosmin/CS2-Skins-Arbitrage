package csfloat

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ListingsService
}

func NewHandler(service ListingsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetListings(c *gin.Context) {
	filters, err := parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.FetchListings(filters)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, ErrMissingAPIKey) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetHistory(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el parámetro 'name' es obligatorio"})
		return
	}

	history, err := h.service.FetchHistory(name)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, ErrMissingAPIKey) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func parseFilters(c *gin.Context) (ListingsFilters, error) {
	minPrice, err := parseFloatQuery(c, "min_price", DefaultMinPrice)
	if err != nil {
		return ListingsFilters{}, err
	}

	maxPrice, err := parseFloatQuery(c, "max_price", DefaultMaxPrice)
	if err != nil {
		return ListingsFilters{}, err
	}

	limit, err := parseIntQuery(c, "limit", DefaultLimit)
	if err != nil {
		return ListingsFilters{}, err
	}
	if limit <= 0 {
		limit = DefaultLimit
	} else if limit > MaxAllowedLimit {
		limit = MaxAllowedLimit
	}

	sort := c.DefaultQuery("sort", DefaultSort)
	onlyNoFactor, err := parseBoolQuery(c, "only_no_factor", true)
	if err != nil {
		return ListingsFilters{}, err
	}

	avoidPanicSells, err := parseBoolQuery(c, "avoid_panic_sells", true)
	if err != nil {
		return ListingsFilters{}, err
	}

	uniquePerSkin, err := parseBoolQuery(c, "unique_per_skin", true)
	if err != nil {
		return ListingsFilters{}, err
	}

	validateHistory, err := parseBoolQuery(c, "validate_history", false)
	if err != nil {
		return ListingsFilters{}, err
	}

	return ListingsFilters{
		MinPrice:        minPrice,
		MaxPrice:        maxPrice,
		Limit:           limit,
		Sort:            sort,
		OnlyNoFactor:    onlyNoFactor,
		AvoidPanicSells: avoidPanicSells,
		UniquePerSkin:   uniquePerSkin,
		ValidateHistory: validateHistory,
	}, nil
}

func parseFloatQuery(c *gin.Context, key string, fallback float64) (float64, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseBoolQuery(c *gin.Context, key string, fallback bool) (bool, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return parsed, nil
}
