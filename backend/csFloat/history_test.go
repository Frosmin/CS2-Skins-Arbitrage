package csfloat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestFetchHistoryReturnsGraphAndSales(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/graph") {
			resp := []GraphPoint{
				{Day: "2026-09-01T00:00:00Z", AvgPrice: 380, Count: 10},
				{Day: "2026-09-02T00:00:00Z", AvgPrice: 375, Count: 8},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if strings.Contains(r.URL.Path, "/sales") {
			resp := []SaleRecord{
				{
					ID:        "sale-1",
					Price:     370,
					SoldAt:    "2026-09-02T12:00:00Z",
					Wear:      0.15,
					IconURL:   "test-icon",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	svc := NewServiceWithBaseURL(server.Client(), server.URL, "test-api-key")

	history, err := svc.FetchHistory("AK-47 | Slate (Field-Tested)")
	if err != nil {
		t.Fatalf("unexpected error fetching history: %v", err)
	}

	if len(history.Graph) != 2 {
		t.Fatalf("expected 2 graph points, got %d", len(history.Graph))
	}
	if history.Graph[0].AvgPriceUSD != 3.80 {
		t.Fatalf("expected first point avg price 3.80, got %.2f", history.Graph[0].AvgPriceUSD)
	}
	if len(history.Sales) != 1 {
		t.Fatalf("expected 1 sale record, got %d", len(history.Sales))
	}
	if history.Sales[0].PriceUSD != 3.70 {
		t.Fatalf("expected sale price 3.70, got %.2f", history.Sales[0].PriceUSD)
	}
}

func TestFetchHistoryUsesCacheWithinTTL(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if strings.Contains(r.URL.Path, "/graph") {
			_ = json.NewEncoder(w).Encode([]GraphPoint{
				{Day: "2026-09-01T00:00:00Z", AvgPrice: 380, Count: 10},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/sales") {
			_ = json.NewEncoder(w).Encode([]SaleRecord{})
			return
		}
	}))
	defer server.Close()

	svc := NewServiceWithBaseURL(server.Client(), server.URL, "test-api-key")

	// Call 1: should make 2 HTTP requests (graph + sales)
	_, err := svc.FetchHistory("M4A4 | Magnesium (Field-Tested)")
	if err != nil {
		t.Fatalf("call 1 failed: %v", err)
	}

	firstCount := atomic.LoadInt32(&requestCount)
	if firstCount != 2 {
		t.Fatalf("expected 2 requests on first fetch, got %d", firstCount)
	}

	// Call 2: should hit in-memory cache and not make any additional HTTP requests
	_, err = svc.FetchHistory("M4A4 | Magnesium (Field-Tested)")
	if err != nil {
		t.Fatalf("call 2 failed: %v", err)
	}

	secondCount := atomic.LoadInt32(&requestCount)
	if secondCount != 2 {
		t.Fatalf("expected cache hit with 2 total requests, got %d", secondCount)
	}
}

func TestHistoryHandlerReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubListingsService{
		historyResponse: HistoryData{
			MarketHashName: "AK-47 | Slate (Field-Tested)",
			Graph: []GraphPoint{
				{Day: "2026-09-01T00:00:00Z", AvgPrice: 380, AvgPriceUSD: 3.80, Count: 5},
			},
			Sales: []SaleRecord{
				{ID: "s1", Price: 380, PriceUSD: 3.80, SoldAt: "2026-09-01T10:00:00Z"},
			},
		},
	}

	router := gin.New()
	router.GET("/api/history", NewHandler(service).GetHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/history?name=AK-47%20%7C%20Slate%20(Field-Tested)", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var res HistoryData
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("error unmarshaling response: %v", err)
	}
	if res.MarketHashName != "AK-47 | Slate (Field-Tested)" {
		t.Fatalf("expected skin name in response, got %s", res.MarketHashName)
	}
}

func TestFetchListingsValidatesHistoryWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/graph") {
			// Historical average is $1.00 (100 cents) over past 7 days
			today := time.Now().UTC()
			resp := []GraphPoint{
				{Day: today.AddDate(0, 0, -1).Format(time.RFC3339), AvgPrice: 100, Count: 5},
				{Day: today.AddDate(0, 0, -2).Format(time.RFC3339), AvgPrice: 100, Count: 5},
				{Day: today.AddDate(0, 0, -3).Format(time.RFC3339), AvgPrice: 100, Count: 5},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if strings.Contains(r.URL.Path, "/sales") {
			_ = json.NewEncoder(w).Encode([]SaleRecord{})
			return
		}

		// Listings endpoint
		resp := csfloatResponse{
			Data: []csfloatListing{
				// Item 1: CSFloat price $0.70 vs base price $1.00 (Steam ref).
				// Historical avg is $1.00. Current price $0.70 is below historical avg -> Valid quicksell!
				{
					ID:        "item-valid",
					Price:     70,
					Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
					Item:      csfloatItem{MarketHashName: "AK-47 | Slate (Field-Tested)"},
				},
				// Item 2: CSFloat price $1.10 vs base price $1.50.
				// Historical avg is $1.00. Current price $1.10 is ABOVE historical avg ($1.00) -> Invalid!
				{
					ID:        "item-invalid",
					Price:     110,
					Reference: csfloatReference{BasePrice: 150, PredictedPrice: 150},
					Item:      csfloatItem{MarketHashName: "USP-S | Ticket to Hell (Field-Tested)"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := NewServiceWithBaseURL(server.Client(), server.URL, "test-api-key")

	res, err := svc.FetchListings(ListingsFilters{
		Sort:            "best_deal",
		ValidateHistory: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Items) != 1 {
		t.Fatalf("expected 1 valid item below historical avg, got %d", len(res.Items))
	}
	if res.Items[0].ID != "item-valid" {
		t.Fatalf("expected item-valid, got %s", res.Items[0].ID)
	}
	if res.Items[0].HistoricalAvg != 1.00 {
		t.Fatalf("expected HistoricalAvg 1.00, got %.2f", res.Items[0].HistoricalAvg)
	}
}
