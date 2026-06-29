export interface ListingsFilters {
  min_price: number;
  max_price: number;
  limit: number;
  sort: string;
  only_no_factor: boolean;
}

export interface ListingOpportunity {
  id: string;
  market_hash_name: string;
  wear: number;
  csfloat_price: number;
  steam_reference_price: number;
  predicted_price: number;
  item_factor: number;
  discount_percent: number;
  purchase_url: string;
}

export interface ListingsResponse {
  items: ListingOpportunity[];
  filters: ListingsFilters;
  count: number;
}

export const DEFAULT_FILTERS: ListingsFilters = {
  min_price: 0.03,
  max_price: 1,
  limit: 50,
  sort: 'best_deal',
  only_no_factor: true,
};
