import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Subject, of, throwError } from 'rxjs';
import { vi } from 'vitest';

import { ListingsApi } from '../../data/listings-api';
import { ListingsResponse } from '../../types/listings';
import { PricesList } from './prices-list';

describe('PricesList', () => {
  let fixture: ComponentFixture<PricesList>;
  let api: { getListings: ReturnType<typeof vi.fn> };

  const response: ListingsResponse = {
    items: [
      {
        id: '1',
        market_hash_name: 'AK-47 | Slate (Field-Tested)',
        wear: 0.15342,
        csfloat_price: 0.82,
        steam_reference_price: 1.14,
        predicted_price: 1.14,
        item_factor: 0,
        discount_percent: 28.07,
        purchase_url: 'https://csfloat.com/item/1',
      },
    ],
    filters: {
      min_price: 0.03,
      max_price: 1,
      limit: 50,
      sort: 'best_deal',
      only_no_factor: true,
    },
    count: 1,
  };

  beforeEach(async () => {
    api = {
      getListings: vi.fn().mockReturnValue(of(response)),
    };

    await TestBed.configureTestingModule({
      imports: [PricesList],
      providers: [{ provide: ListingsApi, useValue: api }],
    }).compileComponents();

    fixture = TestBed.createComponent(PricesList);
  });

  it('loads listings on init with default filters', () => {
    fixture.detectChanges();

    expect(api.getListings).toHaveBeenCalledWith({
      min_price: 0.03,
      max_price: 1,
      limit: 50,
      sort: 'best_deal',
      only_no_factor: true,
    });
  });

  it('renders listing rows when data arrives', () => {
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('AK-47 | Slate (Field-Tested)');
    expect(compiled.textContent).toContain('28.07%');
  });

  it('renders empty state when there are no items', async () => {
    api.getListings.mockReturnValue(
      of({
        ...response,
        items: [],
        count: 0,
      }),
    );

    fixture = TestBed.createComponent(PricesList);
    fixture.detectChanges();
    await fixture.whenStable();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('No encontramos oportunidades');
  });

  it('renders an error message when the request fails', async () => {
    api.getListings.mockReturnValue(
      throwError(() => ({
        error: { error: 'la variable de entorno CSFLOAT_API_KEY está vacía' },
      })),
    );

    fixture = TestBed.createComponent(PricesList);
    fixture.detectChanges();
    await fixture.whenStable();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('CSFLOAT_API_KEY');
  });

  it('shows loading state before the request resolves', () => {
    const pending = new Subject<ListingsResponse>();
    api.getListings.mockReturnValue(pending.asObservable());

    fixture = TestBed.createComponent(PricesList);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Cargando oportunidades...');
  });

  it('renders listings ordered by highest discount percent first by default', () => {
    api.getListings.mockReturnValue(
      of({
        ...response,
        items: [
          {
            id: '1',
            market_hash_name: 'Skin 10 Percent',
            wear: 0.15,
            csfloat_price: 90,
            steam_reference_price: 100,
            predicted_price: 100,
            item_factor: 0,
            discount_percent: 10.0,
            purchase_url: 'https://csfloat.com/item/1',
          },
          {
            id: '2',
            market_hash_name: 'Skin 50 Percent',
            wear: 0.2,
            csfloat_price: 50,
            steam_reference_price: 100,
            predicted_price: 100,
            item_factor: 0,
            discount_percent: 50.0,
            purchase_url: 'https://csfloat.com/item/2',
          },
          {
            id: '3',
            market_hash_name: 'Skin 25 Percent',
            wear: 0.3,
            csfloat_price: 75,
            steam_reference_price: 100,
            predicted_price: 100,
            item_factor: 0,
            discount_percent: 25.0,
            purchase_url: 'https://csfloat.com/item/3',
          },
        ],
        count: 3,
      }),
    );

    fixture = TestBed.createComponent(PricesList);
    fixture.detectChanges();

    const cardNames = Array.from(
      fixture.nativeElement.querySelectorAll('.skin-name') as NodeListOf<HTMLElement>,
    ).map((el) => el.textContent?.trim());

    expect(cardNames).toEqual(['Skin 50 Percent', 'Skin 25 Percent', 'Skin 10 Percent']);
  });

  it('toggles discount sorting when clicking the discount sort button', () => {
    api.getListings.mockReturnValue(
      of({
        ...response,
        items: [
          {
            id: '1',
            market_hash_name: 'Skin 10 Percent',
            wear: 0.15,
            csfloat_price: 90,
            steam_reference_price: 100,
            predicted_price: 100,
            item_factor: 0,
            discount_percent: 10.0,
            purchase_url: 'https://csfloat.com/item/1',
          },
          {
            id: '2',
            market_hash_name: 'Skin 50 Percent',
            wear: 0.2,
            csfloat_price: 50,
            steam_reference_price: 100,
            predicted_price: 100,
            item_factor: 0,
            discount_percent: 50.0,
            purchase_url: 'https://csfloat.com/item/2',
          },
        ],
        count: 2,
      }),
    );

    fixture = TestBed.createComponent(PricesList);
    fixture.detectChanges();

    const sortButton = fixture.nativeElement.querySelector('.sort-toggle-button') as HTMLButtonElement;
    expect(sortButton).toBeTruthy();

    let cardNames = Array.from(
      fixture.nativeElement.querySelectorAll('.skin-name') as NodeListOf<HTMLElement>,
    ).map((el) => el.textContent?.trim());
    expect(cardNames).toEqual(['Skin 50 Percent', 'Skin 10 Percent']);

    sortButton.click();
    fixture.detectChanges();

    cardNames = Array.from(
      fixture.nativeElement.querySelectorAll('.skin-name') as NodeListOf<HTMLElement>,
    ).map((el) => el.textContent?.trim());
    expect(cardNames).toEqual(['Skin 10 Percent', 'Skin 50 Percent']);

    sortButton.click();
    fixture.detectChanges();

    cardNames = Array.from(
      fixture.nativeElement.querySelectorAll('.skin-name') as NodeListOf<HTMLElement>,
    ).map((el) => el.textContent?.trim());
    expect(cardNames).toEqual(['Skin 50 Percent', 'Skin 10 Percent']);
  });
});
