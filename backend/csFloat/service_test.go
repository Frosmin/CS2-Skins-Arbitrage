package csfloat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMapListingOpportunityUsesPredictedPriceAndDiscount(t *testing.T) {
	listing := csfloatListing{
		ID:    "123",
		Price: 82,
		Reference: csfloatReference{
			BasePrice:      114,
			PredictedPrice: 140,
		},
		Item: csfloatItem{
			MarketHashName: "AK-47 | Slate (Field-Tested)",
			Wear:           0.15342,
		},
	}

	opportunity, ok := mapListingOpportunity(listing, ListingsFilters{OnlyNoFactor: false})
	if !ok {
		t.Fatalf("expected listing to be accepted")
	}

	if opportunity.ItemFactor != 0.26 {
		t.Fatalf("expected item factor 0.26, got %.2f", opportunity.ItemFactor)
	}

	if opportunity.PredictedPrice != 1.40 {
		t.Fatalf("expected predicted price 1.40, got %.2f", opportunity.PredictedPrice)
	}

	if opportunity.DiscountPercent != 28.07 {
		t.Fatalf("expected discount 28.07, got %.2f", opportunity.DiscountPercent)
	}
}

func TestMapListingOpportunityFallsBackToBasePriceWhenPredictedMissing(t *testing.T) {
	listing := csfloatListing{
		ID:    "456",
		Price: 75,
		Reference: csfloatReference{
			BasePrice:      100,
			PredictedPrice: 0,
		},
		Item: csfloatItem{
			MarketHashName: "M4A4 | Magnesium",
			Wear:           0.20123,
		},
	}

	opportunity, ok := mapListingOpportunity(listing, ListingsFilters{OnlyNoFactor: false})
	if !ok {
		t.Fatalf("expected listing to be accepted")
	}

	if opportunity.PredictedPrice != 1.00 {
		t.Fatalf("expected predicted price 1.00, got %.2f", opportunity.PredictedPrice)
	}

	if opportunity.ItemFactor != 0 {
		t.Fatalf("expected item factor 0, got %.2f", opportunity.ItemFactor)
	}
}

func TestMapListingOpportunityKeepsNegativeFactorWhenOnlyNoFactorIsEnabled(t *testing.T) {
	listing := csfloatListing{
		ID:    "789",
		Price: 70,
		Reference: csfloatReference{
			BasePrice:      100,
			PredictedPrice: 90,
		},
		Item: csfloatItem{
			MarketHashName: "USP-S | Ticket to Hell",
			Wear:           0.09012,
		},
	}

	_, ok := mapListingOpportunity(listing, ListingsFilters{OnlyNoFactor: true})
	if !ok {
		t.Fatalf("expected negative item factor listing to remain visible")
	}
}

func TestMapListingOpportunityFiltersPositiveFactorWhenOnlyNoFactorIsEnabled(t *testing.T) {
	listing := csfloatListing{
		ID:    "789",
		Price: 70,
		Reference: csfloatReference{
			BasePrice:      100,
			PredictedPrice: 125,
		},
		Item: csfloatItem{
			MarketHashName: "USP-S | Ticket to Hell",
			Wear:           0.09012,
		},
	}

	_, ok := mapListingOpportunity(listing, ListingsFilters{OnlyNoFactor: true})
	if ok {
		t.Fatalf("expected listing to be filtered out when item factor is positive")
	}
}

func TestMapListingOpportunityMapsIconURL(t *testing.T) {
	tests := []struct {
		name     string
		rawIcon  string
		expected string
	}{
		{
			name:     "Steam icon hash",
			rawIcon:  "-9a81dlWLwJ2UUGcVs_nsVtzdOEdtWwKGZZLQHTxDZ7I56KU0Zwwo4NUX4oFJZEHLbXH5ApeO4YmlhxYQknCRvCo04DEVlxkKgpou-6kejhjxszYfi5H5di5mr-HnvD8J_WCkmkEvp0pi7zDodv3jAHj-UM5ZGr7INfHJAc9MlzV-FK_kO281pa_ot2XnrA-A3kA",
			expected: "https://community.cloudflare.steamstatic.com/economy/image/-9a81dlWLwJ2UUGcVs_nsVtzdOEdtWwKGZZLQHTxDZ7I56KU0Zwwo4NUX4oFJZEHLbXH5ApeO4YmlhxYQknCRvCo04DEVlxkKgpou-6kejhjxszYfi5H5di5mr-HnvD8J_WCkmkEvp0pi7zDodv3jAHj-UM5ZGr7INfHJAc9MlzV-FK_kO281pa_ot2XnrA-A3kA",
		},
		{
			name:     "Already full URL",
			rawIcon:  "https://example.com/weapon.png",
			expected: "https://example.com/weapon.png",
		},
		{
			name:     "Empty icon",
			rawIcon:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listing := csfloatListing{
				ID:    "111",
				Price: 50,
				Reference: csfloatReference{
					BasePrice:      100,
					PredictedPrice: 100,
				},
				Item: csfloatItem{
					MarketHashName: "AK-47 | Redline (Field-Tested)",
					Wear:           0.22,
					IconURL:        tt.rawIcon,
				},
			}

			opportunity, ok := mapListingOpportunity(listing, ListingsFilters{OnlyNoFactor: false})
			if !ok {
				t.Fatalf("expected listing to be valid")
			}
			if opportunity.IconURL != tt.expected {
				t.Fatalf("expected icon_url '%s', got '%s'", tt.expected, opportunity.IconURL)
			}
		})
	}
}

func TestFetchListingsOrdersByHighestDiscountFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := csfloatResponse{
			Data: []csfloatListing{
				{
					ID:    "item-1",
					Price: 90,
					Reference: csfloatReference{
						BasePrice:      100,
						PredictedPrice: 100,
					},
					Item: csfloatItem{MarketHashName: "Skin 1"},
				},
				{
					ID:    "item-2",
					Price: 50,
					Reference: csfloatReference{
						BasePrice:      100,
						PredictedPrice: 100,
					},
					Item: csfloatItem{MarketHashName: "Skin 2"},
				},
				{
					ID:    "item-3",
					Price: 75,
					Reference: csfloatReference{
						BasePrice:      100,
						PredictedPrice: 100,
					},
					Item: csfloatItem{MarketHashName: "Skin 3"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := &Service{
		client:  server.Client(),
		baseURL: server.URL,
		apiKey:  "test-api-key",
	}

	res, err := svc.FetchListings(ListingsFilters{Sort: "best_deal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res.Items))
	}

	if res.Items[0].ID != "item-2" || res.Items[0].DiscountPercent != 50.0 {
		t.Fatalf("expected item-2 with 50%% discount first, got %s (%.2f%%)", res.Items[0].ID, res.Items[0].DiscountPercent)
	}
	if res.Items[1].ID != "item-3" || res.Items[1].DiscountPercent != 25.0 {
		t.Fatalf("expected item-3 with 25%% discount second, got %s (%.2f%%)", res.Items[1].ID, res.Items[1].DiscountPercent)
	}
	if res.Items[2].ID != "item-1" || res.Items[2].DiscountPercent != 10.0 {
		t.Fatalf("expected item-1 with 10%% discount third, got %s (%.2f%%)", res.Items[2].ID, res.Items[2].DiscountPercent)
	}
}

func TestFetchListingsPaginatesWhenLimitExceeds50(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		limit := r.URL.Query().Get("limit")
		cursor := r.URL.Query().Get("cursor")

		if limit != "50" {
			t.Errorf("expected batch limit 50, got %s", limit)
		}

		if requestCount == 1 {
			if cursor != "" {
				t.Errorf("expected empty cursor on first request, got %s", cursor)
			}
			resp := csfloatResponse{
				Cursor: "next-cursor-token",
				Data: []csfloatListing{
					{
						ID:    "item-1",
						Price: 50,
						Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
						Item: csfloatItem{MarketHashName: "Skin 1"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if cursor != "next-cursor-token" {
			t.Errorf("expected cursor 'next-cursor-token' on second request, got %s", cursor)
		}
		resp := csfloatResponse{
			Cursor: "",
			Data: []csfloatListing{
				{
					ID:    "item-2",
					Price: 60,
					Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
					Item: csfloatItem{MarketHashName: "Skin 2"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := &Service{
		client:  server.Client(),
		baseURL: server.URL,
		apiKey:  "test-api-key",
	}

	res, err := svc.FetchListings(ListingsFilters{Limit: 100, Sort: "best_deal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}

	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
}

func TestFetchListingsFiltersOutPanicSells(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := csfloatResponse{
			Data: []csfloatListing{
				{
					ID:        "ump-1",
					Price:     29,
					Reference: csfloatReference{BasePrice: 36, PredictedPrice: 36},
					Item:      csfloatItem{MarketHashName: "UMP-45 | Labyrinth (Minimal Wear)"},
				},
				{
					ID:        "ump-2",
					Price:     29,
					Reference: csfloatReference{BasePrice: 36, PredictedPrice: 36},
					Item:      csfloatItem{MarketHashName: "UMP-45 | Labyrinth (Minimal Wear)"},
				},
				{
					ID:        "ump-3",
					Price:     29,
					Reference: csfloatReference{BasePrice: 36, PredictedPrice: 36},
					Item:      csfloatItem{MarketHashName: "UMP-45 | Labyrinth (Minimal Wear)"},
				},
				{
					ID:        "ak-1",
					Price:     50,
					Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
					Item:      csfloatItem{MarketHashName: "AK-47 | Slate (Field-Tested)"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := &Service{
		client:  server.Client(),
		baseURL: server.URL,
		apiKey:  "test-api-key",
	}

	res, err := svc.FetchListings(ListingsFilters{
		Sort:            "best_deal",
		AvoidPanicSells: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Items) != 1 {
		t.Fatalf("expected 1 item (only the isolated quicksell), got %d", len(res.Items))
	}
	if res.Items[0].MarketHashName != "AK-47 | Slate (Field-Tested)" {
		t.Fatalf("expected AK-47, got %s", res.Items[0].MarketHashName)
	}
}

func TestFetchListingsDeduplicatesSameSkin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := csfloatResponse{
			Data: []csfloatListing{
				{
					ID:        "mag-1",
					Price:     75,
					Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
					Item:      csfloatItem{MarketHashName: "M4A4 | Magnesium"},
				},
				{
					ID:        "mag-2",
					Price:     50,
					Reference: csfloatReference{BasePrice: 100, PredictedPrice: 100},
					Item:      csfloatItem{MarketHashName: "M4A4 | Magnesium"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := &Service{
		client:  server.Client(),
		baseURL: server.URL,
		apiKey:  "test-api-key",
	}

	res, err := svc.FetchListings(ListingsFilters{
		Sort:          "best_deal",
		UniquePerSkin: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Items) != 1 {
		t.Fatalf("expected 1 deduplicated item, got %d", len(res.Items))
	}
	if res.Items[0].ID != "mag-2" || res.Items[0].CSFloatPrice != 0.50 {
		t.Fatalf("expected best deal (mag-2 @ $0.50), got %s @ $%.2f", res.Items[0].ID, res.Items[0].CSFloatPrice)
	}
}

