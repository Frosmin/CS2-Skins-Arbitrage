package csfloat

import "testing"

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

func TestMapListingOpportunityFiltersNoFactor(t *testing.T) {
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
		t.Fatalf("expected listing to be filtered out when item factor is present")
	}
}
