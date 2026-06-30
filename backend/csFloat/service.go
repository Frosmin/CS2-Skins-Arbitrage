package csfloat

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultMinPrice = 0.03
	DefaultMaxPrice = 1.00
	DefaultLimit    = 50
	DefaultSort     = "best_deal"
)

var ErrMissingAPIKey = errors.New("la variable de entorno CSFLOAT_API_KEY está vacía")

type ListingsFilters struct {
	MinPrice     float64 `json:"min_price"`
	MaxPrice     float64 `json:"max_price"`
	Limit        int     `json:"limit"`
	Sort         string  `json:"sort"`
	OnlyNoFactor bool    `json:"only_no_factor"`
}

type ListingOpportunity struct {
	ID                  string  `json:"id"`
	MarketHashName      string  `json:"market_hash_name"`
	Wear                float64 `json:"wear"`
	CSFloatPrice        float64 `json:"csfloat_price"`
	SteamReferencePrice float64 `json:"steam_reference_price"`
	PredictedPrice      float64 `json:"predicted_price"`
	ItemFactor          float64 `json:"item_factor"`
	DiscountPercent     float64 `json:"discount_percent"`
	PurchaseURL         string  `json:"purchase_url"`
}

type ListingsResponse struct {
	Items   []ListingOpportunity `json:"items"`
	Filters ListingsFilters      `json:"filters"`
	Count   int                  `json:"count"`
}

type ListingsService interface {
	FetchListings(filters ListingsFilters) (ListingsResponse, error)
}

type Service struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

type csfloatResponse struct {
	Data []csfloatListing `json:"data"`
}

type csfloatListing struct {
	ID        string           `json:"id"`
	Price     int64            `json:"price"`
	Reference csfloatReference `json:"reference"`
	Item      csfloatItem      `json:"item"`
}

type csfloatReference struct {
	BasePrice      int64 `json:"base_price"`
	PredictedPrice int64 `json:"predicted_price"`
}

type csfloatItem struct {
	MarketHashName string  `json:"market_hash_name"`
	Wear           float64 `json:"float_value"`
}

func NewService(client *http.Client) *Service {
	_ = godotenv.Load()
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &Service{
		client:  client,
		baseURL: "https://csfloat.com/api/v1/listings",
		apiKey:  os.Getenv("CSFLOAT_API_KEY"),
	}
}

func (s *Service) FetchListings(filters ListingsFilters) (ListingsResponse, error) {
	if s.apiKey == "" {
		return ListingsResponse{}, ErrMissingAPIKey
	}

	requestURL, err := buildListingsURL(s.baseURL, filters)
	if err != nil {
		return ListingsResponse{}, err
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return ListingsResponse{}, fmt.Errorf("error al crear la petición: %w", err)
	}

	req.Header.Set("Authorization", s.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CS2-Arbitrage-App/2.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return ListingsResponse{}, fmt.Errorf("error de conexión con CSFloat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ListingsResponse{}, fmt.Errorf("error en la API de CSFloat. Código de estado: %d", resp.StatusCode)
	}

	var payload csfloatResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ListingsResponse{}, fmt.Errorf("error al decodificar respuesta de CSFloat: %w", err)
	}

	items := make([]ListingOpportunity, 0, len(payload.Data))
	for _, listing := range payload.Data {
		opportunity, ok := mapListingOpportunity(listing, filters)
		if ok {
			items = append(items, opportunity)
		}
	}

	return ListingsResponse{
		Items:   items,
		Filters: filters,
		Count:   len(items),
	}, nil
}

func buildListingsURL(baseURL string, filters ListingsFilters) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("error parseando BaseURL: %w", err)
	}

	query := parsedURL.Query()
	query.Set("limit", strconv.Itoa(filters.Limit))
	query.Set("sort_by", filters.Sort)
	query.Set("category", "1")
	query.Set("type", "buy_now")
	query.Set("min_price", strconv.FormatInt(int64(filters.MinPrice*100), 10))
	query.Set("max_price", strconv.FormatInt(int64(filters.MaxPrice*100), 10))
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func mapListingOpportunity(listing csfloatListing, filters ListingsFilters) (ListingOpportunity, bool) {
	steamReferencePrice := centsToUSD(listing.Reference.BasePrice)
	csfloatPrice := centsToUSD(listing.Price)
	predictedPriceCents := listing.Reference.PredictedPrice
	if predictedPriceCents <= 0 {
		predictedPriceCents = listing.Reference.BasePrice
	}

	itemFactorCents := predictedPriceCents - listing.Reference.BasePrice
	if filters.OnlyNoFactor && itemFactorCents > 0 {
		return ListingOpportunity{}, false
	}

	if steamReferencePrice <= 0 || steamReferencePrice <= csfloatPrice {
		return ListingOpportunity{}, false
	}

	discountPercent := ((steamReferencePrice - csfloatPrice) / steamReferencePrice) * 100

	return ListingOpportunity{
		ID:                  listing.ID,
		MarketHashName:      listing.Item.MarketHashName,
		Wear:                listing.Item.Wear,
		CSFloatPrice:        roundToTwo(csfloatPrice),
		SteamReferencePrice: roundToTwo(steamReferencePrice),
		PredictedPrice:      roundToTwo(centsToUSD(predictedPriceCents)),
		ItemFactor:          roundToTwo(centsToUSD(itemFactorCents)),
		DiscountPercent:     roundToTwo(discountPercent),
		PurchaseURL:         fmt.Sprintf("https://csfloat.com/item/%s", listing.ID),
	}, true
}

func centsToUSD(cents int64) float64 {
	return float64(cents) / 100.0
}

func roundToTwo(value float64) float64 {
	return math.Round(value*100) / 100
}
