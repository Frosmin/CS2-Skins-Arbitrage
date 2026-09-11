export interface ListingsFilters {
  min_price: number;
  max_price: number;
  limit: number;
  sort: string;
  only_no_factor: boolean;
  avoid_panic_sells: boolean;
  unique_per_skin: boolean;
  validate_history: boolean;
}

export interface ListingOpportunity {
  id: string;
  market_hash_name: string;
  wear: number;
  icon_url?: string;
  csfloat_price: number;
  steam_reference_price: number;
  predicted_price: number;
  item_factor: number;
  discount_percent: number;
  historical_avg?: number;
  purchase_url: string;
}

export interface HistoryGraphPoint {
  day: string;
  avg_price: number;
  avg_price_usd: number;
  count: number;
}

export interface RecentSaleRecord {
  id: string;
  price: number;
  price_usd: number;
  sold_at: string;
  wear: number;
  icon_url: string;
}

export interface SkinHistoryResponse {
  market_hash_name: string;
  graph: HistoryGraphPoint[];
  sales: RecentSaleRecord[];
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
  avoid_panic_sells: true,
  unique_per_skin: true,
  validate_history: false,
};
