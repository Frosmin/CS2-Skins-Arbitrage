package csfloat

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubListingsService struct {
	filters  ListingsFilters
	response ListingsResponse
	err      error
}

func (s *stubListingsService) FetchListings(filters ListingsFilters) (ListingsResponse, error) {
	s.filters = filters
	if s.err != nil {
		return ListingsResponse{}, s.err
	}
	return s.response, nil
}

func TestListingsHandlerUsesDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubListingsService{
		response: ListingsResponse{
			Items: []ListingOpportunity{},
		},
	}
	router := gin.New()
	router.GET("/api/listings", NewHandler(service).GetListings)

	req := httptest.NewRequest(http.MethodGet, "/api/listings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if service.filters.MinPrice != DefaultMinPrice {
		t.Fatalf("expected default min price %.2f, got %.2f", DefaultMinPrice, service.filters.MinPrice)
	}

	if service.filters.MaxPrice != DefaultMaxPrice {
		t.Fatalf("expected default max price %.2f, got %.2f", DefaultMaxPrice, service.filters.MaxPrice)
	}

	if !service.filters.OnlyNoFactor {
		t.Fatalf("expected only_no_factor default true")
	}
}

func TestListingsHandlerRejectsInvalidParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/listings", NewHandler(&stubListingsService{}).GetListings)

	req := httptest.NewRequest(http.MethodGet, "/api/listings?limit=nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestListingsHandlerMapsMissingAPIKeyErrorTo500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/listings", NewHandler(&stubListingsService{
		err: ErrMissingAPIKey,
	}).GetListings)

	req := httptest.NewRequest(http.MethodGet, "/api/listings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestListingsHandlerMapsUpstreamErrorsTo502(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/listings", NewHandler(&stubListingsService{
		err: errors.New("upstream failed"),
	}).GetListings)

	req := httptest.NewRequest(http.MethodGet, "/api/listings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}
}
