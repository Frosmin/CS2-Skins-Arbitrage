package csfloat

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

const steamImageBaseURL = "https://community.cloudflare.steamstatic.com/economy/image/"

const (
	DefaultMinPrice = 0.03
	DefaultMaxPrice = 1.00
	DefaultLimit    = 50
	MaxAllowedLimit = 100000
	CSFloatMaxBatch = 50
	DefaultSort     = "best_deal"
	HistoryCacheTTL = 30 * time.Minute
)

var ErrMissingAPIKey = errors.New("la variable de entorno CSFLOAT_API_KEY está vacía")

type ListingsFilters struct {
	MinPrice        float64 `json:"min_price"`
	MaxPrice        float64 `json:"max_price"`
	Limit           int     `json:"limit"`
	Sort            string  `json:"sort"`
	OnlyNoFactor    bool    `json:"only_no_factor"`
	AvoidPanicSells bool    `json:"avoid_panic_sells"`
	UniquePerSkin   bool    `json:"unique_per_skin"`
	ValidateHistory bool    `json:"validate_history"`
}

type ListingOpportunity struct {
	ID                  string  `json:"id"`
	MarketHashName      string  `json:"market_hash_name"`
	Wear                float64 `json:"wear"`
	IconURL             string  `json:"icon_url"`
	CSFloatPrice        float64 `json:"csfloat_price"`
	SteamReferencePrice float64 `json:"steam_reference_price"`
	PredictedPrice      float64 `json:"predicted_price"`
	ItemFactor          float64 `json:"item_factor"`
	DiscountPercent     float64 `json:"discount_percent"`
	HistoricalAvg       float64 `json:"historical_avg,omitempty"`
	PurchaseURL         string  `json:"purchase_url"`
}

type GraphPoint struct {
	Day         string  `json:"day"`
	AvgPrice    float64 `json:"avg_price"`
	AvgPriceUSD float64 `json:"avg_price_usd"`
	Count       int     `json:"count"`
}

type SaleRecord struct {
	ID       string  `json:"id"`
	Price    int64   `json:"price"`
	PriceUSD float64 `json:"price_usd"`
	SoldAt   string  `json:"sold_at"`
	Wear     float64 `json:"wear"`
	IconURL  string  `json:"icon_url"`
}

type rawSaleItem struct {
	ID     string `json:"id"`
	Price  int64  `json:"price"`
	SoldAt string `json:"sold_at"`
	Item   struct {
		FloatValue float64 `json:"float_value"`
		IconURL    string  `json:"icon_url"`
	} `json:"item"`
}

type HistoryData struct {
	MarketHashName string       `json:"market_hash_name"`
	Graph          []GraphPoint `json:"graph"`
	Sales          []SaleRecord `json:"sales"`
}

type cacheEntry struct {
	data      HistoryData
	expiresAt time.Time
}

type ListingsResponse struct {
	Items   []ListingOpportunity `json:"items"`
	Filters ListingsFilters      `json:"filters"`
	Count   int                  `json:"count"`
}

type ListingsService interface {
	FetchListings(filters ListingsFilters) (ListingsResponse, error)
	FetchHistory(name string) (HistoryData, error)
}

type Service struct {
	client     *http.Client
	baseURL    string
	historyURL string
	apiKey     string
	cache      map[string]cacheEntry
	cacheMu    sync.RWMutex
}

type csfloatResponse struct {
	Data   []csfloatListing `json:"data"`
	Cursor string           `json:"cursor"`
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
	IconURL        string  `json:"icon_url"`
}

func NewService(client *http.Client) *Service {
	_ = godotenv.Load()
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &Service{
		client:     client,
		baseURL:    "https://csfloat.com/api/v1/listings",
		historyURL: "https://csfloat.com/api/v1/history",
		apiKey:     os.Getenv("CSFLOAT_API_KEY"),
		cache:      make(map[string]cacheEntry),
	}
}

func NewServiceWithBaseURL(client *http.Client, baseURL, apiKey string) *Service {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	historyURL := baseURL
	if !strings.Contains(baseURL, "/history") {
		historyURL = baseURL + "/history"
	}
	return &Service{
		client:     client,
		baseURL:    baseURL,
		historyURL: historyURL,
		apiKey:     apiKey,
		cache:      make(map[string]cacheEntry),
	}
}

func (s *Service) FetchHistory(name string) (HistoryData, error) {
	if s.apiKey == "" {
		return HistoryData{}, ErrMissingAPIKey
	}

	s.cacheMu.RLock()
	if entry, ok := s.cache[name]; ok && time.Now().Before(entry.expiresAt) {
		s.cacheMu.RUnlock()
		return entry.data, nil
	}
	s.cacheMu.RUnlock()

	encodedName := url.PathEscape(name)

	graphURL := fmt.Sprintf("%s/%s/graph", s.historyURL, encodedName)
	graphReq, err := http.NewRequest(http.MethodGet, graphURL, nil)
	if err != nil {
		return HistoryData{}, fmt.Errorf("error al crear petición de gráfica: %w", err)
	}
	graphReq.Header.Set("Authorization", s.apiKey)
	graphReq.Header.Set("Accept", "application/json")
	graphReq.Header.Set("User-Agent", "CS2-Arbitrage-App/2.0")

	graphResp, err := s.client.Do(graphReq)
	if err != nil {
		return HistoryData{}, fmt.Errorf("error al conectar con API de historial (gráfica): %w", err)
	}
	defer graphResp.Body.Close()

	var rawGraph []GraphPoint
	if graphResp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(graphResp.Body).Decode(&rawGraph)
	}

	for i := range rawGraph {
		rawGraph[i].AvgPriceUSD = roundToTwo(rawGraph[i].AvgPrice / 100.0)
	}

	salesURL := fmt.Sprintf("%s/%s/sales", s.historyURL, encodedName)
	salesReq, err := http.NewRequest(http.MethodGet, salesURL, nil)
	if err != nil {
		return HistoryData{}, fmt.Errorf("error al crear petición de ventas: %w", err)
	}
	salesReq.Header.Set("Authorization", s.apiKey)
	salesReq.Header.Set("Accept", "application/json")
	salesReq.Header.Set("User-Agent", "CS2-Arbitrage-App/2.0")

	salesResp, err := s.client.Do(salesReq)
	if err != nil {
		return HistoryData{}, fmt.Errorf("error al conectar con API de historial (ventas): %w", err)
	}
	defer salesResp.Body.Close()

	var rawSales []rawSaleItem
	if salesResp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(salesResp.Body).Decode(&rawSales)
	}

	sales := make([]SaleRecord, 0, len(rawSales))
	for _, item := range rawSales {
		sales = append(sales, SaleRecord{
			ID:       item.ID,
			Price:    item.Price,
			PriceUSD: roundToTwo(centsToUSD(item.Price)),
			SoldAt:   item.SoldAt,
			Wear:     item.Item.FloatValue,
			IconURL:  buildItemImageURL(item.Item.IconURL),
		})
	}

	data := HistoryData{
		MarketHashName: name,
		Graph:          rawGraph,
		Sales:          sales,
	}

	s.cacheMu.Lock()
	s.cache[name] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(HistoryCacheTTL),
	}
	s.cacheMu.Unlock()

	return data, nil
}

func (s *Service) calculateRecentAverage(name string) (float64, bool) {
	history, err := s.FetchHistory(name)
	if err != nil || len(history.Graph) == 0 {
		return 0, false
	}
	cutoff := time.Now().AddDate(0, 0, -7)
	var sum float64
	var count int
	for _, pt := range history.Graph {
		t, err := time.Parse(time.RFC3339, pt.Day)
		if err == nil && t.After(cutoff) && pt.AvgPriceUSD > 0 {
			sum += pt.AvgPriceUSD
			count++
		}
	}
	if count == 0 {
		for i := len(history.Graph) - 1; i >= 0 && count < 3; i-- {
			if history.Graph[i].AvgPriceUSD > 0 {
				sum += history.Graph[i].AvgPriceUSD
				count++
			}
		}
	}
	if count == 0 {
		return 0, false
	}
	return roundToTwo(sum / float64(count)), true
}

func (s *Service) FetchListings(filters ListingsFilters) (ListingsResponse, error) {
	if s.apiKey == "" {
		return ListingsResponse{}, ErrMissingAPIKey
	}

	targetLimit := filters.Limit
	if targetLimit <= 0 {
		targetLimit = DefaultLimit
	} else if targetLimit > MaxAllowedLimit {
		targetLimit = MaxAllowedLimit
	}

	var rawListings []csfloatListing
	cursor := ""
	totalRawFetched := 0
	maxRawLimit := targetLimit
	if filters.AvoidPanicSells || filters.UniquePerSkin {
		maxRawLimit = targetLimit * 3
		if maxRawLimit < CSFloatMaxBatch*2 {
			maxRawLimit = CSFloatMaxBatch * 2
		}
		if maxRawLimit > MaxAllowedLimit*2 {
			maxRawLimit = MaxAllowedLimit * 2
		}
	}

	for totalRawFetched < maxRawLimit {
		batchLimit := CSFloatMaxBatch
		if !filters.AvoidPanicSells && !filters.UniquePerSkin {
			remaining := targetLimit - totalRawFetched
			if remaining < batchLimit {
				batchLimit = remaining
			}
		}

		requestURL, err := buildListingsURL(s.baseURL, filters, batchLimit, cursor)
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

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return ListingsResponse{}, fmt.Errorf("error en la API de CSFloat. Código de estado: %d", resp.StatusCode)
		}

		var payload csfloatResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if decodeErr != nil {
			return ListingsResponse{}, fmt.Errorf("error al decodificar respuesta de CSFloat: %w", decodeErr)
		}

		if len(payload.Data) == 0 {
			break
		}

		totalRawFetched += len(payload.Data)
		rawListings = append(rawListings, payload.Data...)

		processed := processAndFilterListings(rawListings, filters)
		if len(processed) >= targetLimit {
			break
		}

		if payload.Cursor == "" || payload.Cursor == cursor {
			break
		}
		cursor = payload.Cursor
	}

	items := processAndFilterListings(rawListings, filters)

	if filters.Sort == "best_deal" || filters.Sort == "" {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].DiscountPercent > items[j].DiscountPercent
		})
	}

	if filters.ValidateHistory && len(items) > 0 {
		validated := make([]ListingOpportunity, 0, len(items))
		topCandidates := items
		if len(topCandidates) > 10 {
			topCandidates = topCandidates[:10]
		}

		for _, opp := range topCandidates {
			avg7d, ok := s.calculateRecentAverage(opp.MarketHashName)
			if ok {
				if opp.CSFloatPrice >= avg7d {
					continue
				}
				opp.HistoricalAvg = avg7d
			}
			validated = append(validated, opp)
		}
		items = validated
	}

	if len(items) > targetLimit {
		items = items[:targetLimit]
	}

	return ListingsResponse{
		Items:   items,
		Filters: filters,
		Count:   len(items),
	}, nil
}

type skinGroup struct {
	minPrice      int64
	minPriceCount int
	opportunities []ListingOpportunity
}

func processAndFilterListings(rawListings []csfloatListing, filters ListingsFilters) []ListingOpportunity {
	if !filters.AvoidPanicSells && !filters.UniquePerSkin {
		items := make([]ListingOpportunity, 0, len(rawListings))
		for _, listing := range rawListings {
			opp, ok := mapListingOpportunity(listing, filters)
			if ok {
				items = append(items, opp)
			}
		}
		return items
	}

	groups := make(map[string]*skinGroup)
	order := make([]string, 0)

	for _, listing := range rawListings {
		opp, ok := mapListingOpportunity(listing, filters)
		if !ok {
			continue
		}

		name := opp.MarketHashName
		group, exists := groups[name]
		if !exists {
			group = &skinGroup{
				minPrice:      listing.Price,
				minPriceCount: 1,
				opportunities: []ListingOpportunity{opp},
			}
			groups[name] = group
			order = append(order, name)
		} else {
			group.opportunities = append(group.opportunities, opp)
			if listing.Price < group.minPrice {
				group.minPrice = listing.Price
				group.minPriceCount = 1
			} else if listing.Price == group.minPrice {
				group.minPriceCount++
			}
		}
	}

	result := make([]ListingOpportunity, 0, len(groups))
	for _, name := range order {
		group := groups[name]

		if filters.AvoidPanicSells && group.minPriceCount >= 3 {
			continue
		}

		if filters.UniquePerSkin {
			best := group.opportunities[0]
			for _, opp := range group.opportunities[1:] {
				if opp.DiscountPercent > best.DiscountPercent ||
					(opp.DiscountPercent == best.DiscountPercent && opp.CSFloatPrice < best.CSFloatPrice) {
					best = opp
				}
			}
			result = append(result, best)
		} else {
			result = append(result, group.opportunities...)
		}
	}

	return result
}

func buildListingsURL(baseURL string, filters ListingsFilters, batchLimit int, cursor string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("error parseando BaseURL: %w", err)
	}

	query := parsedURL.Query()
	query.Set("limit", strconv.Itoa(batchLimit))
	query.Set("sort_by", filters.Sort)
	query.Set("category", "1")
	query.Set("type", "buy_now")
	query.Set("min_price", strconv.FormatInt(int64(filters.MinPrice*100), 10))
	query.Set("max_price", strconv.FormatInt(int64(filters.MaxPrice*100), 10))
	if cursor != "" {
		query.Set("cursor", cursor)
	}
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
		IconURL:             buildItemImageURL(listing.Item.IconURL),
		CSFloatPrice:        roundToTwo(csfloatPrice),
		SteamReferencePrice: roundToTwo(steamReferencePrice),
		PredictedPrice:      roundToTwo(centsToUSD(predictedPriceCents)),
		ItemFactor:          roundToTwo(centsToUSD(itemFactorCents)),
		DiscountPercent:     roundToTwo(discountPercent),
		PurchaseURL:         fmt.Sprintf("https://csfloat.com/item/%s", listing.ID),
	}, true
}

func buildItemImageURL(iconURL string) string {
	if iconURL == "" {
		return ""
	}
	if strings.HasPrefix(iconURL, "http://") || strings.HasPrefix(iconURL, "https://") {
		return iconURL
	}
	return steamImageBaseURL + iconURL
}

func centsToUSD(cents int64) float64 {
	return float64(cents) / 100.0
}

func roundToTwo(value float64) float64 {
	return math.Round(value*100) / 100
}
