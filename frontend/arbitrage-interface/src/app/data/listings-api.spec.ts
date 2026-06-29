import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';

import { ListingsApi } from './listings-api';

describe('ListingsApi', () => {
  let service: ListingsApi;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), ListingsApi],
    });

    service = TestBed.inject(ListingsApi);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('serializes listing filters as query params', () => {
    service
      .getListings({
        min_price: 0.03,
        max_price: 1,
        limit: 50,
        sort: 'best_deal',
        only_no_factor: true,
      })
      .subscribe();

    const request = httpMock.expectOne((req) => req.url === 'http://localhost:8080/api/listings');

    expect(request.request.params.get('min_price')).toBe('0.03');
    expect(request.request.params.get('max_price')).toBe('1');
    expect(request.request.params.get('limit')).toBe('50');
    expect(request.request.params.get('sort')).toBe('best_deal');
    expect(request.request.params.get('only_no_factor')).toBe('true');

    request.flush({ items: [], filters: {}, count: 0 });
  });
});
