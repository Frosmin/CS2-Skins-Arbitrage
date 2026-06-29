import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { ListingsFilters, ListingsResponse } from '../types/listings';

@Injectable({ providedIn: 'root' })
export class ListingsApi {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = 'http://localhost:8080/api/listings';

  getListings(filters: ListingsFilters) {
    const params = new HttpParams({
      fromObject: {
        min_price: String(filters.min_price),
        max_price: String(filters.max_price),
        limit: String(filters.limit),
        sort: filters.sort,
        only_no_factor: String(filters.only_no_factor),
      },
    });

    return this.http.get<ListingsResponse>(this.baseUrl, { params });
  }
}
