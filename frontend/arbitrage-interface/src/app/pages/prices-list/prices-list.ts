import { DecimalPipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ListingsApi } from '../../data/listings-api';
import {
  DEFAULT_FILTERS,
  ListingOpportunity,
  ListingsFilters,
} from '../../types/listings';

@Component({
  selector: 'app-prices-list',
  imports: [FormsModule, DecimalPipe],
  templateUrl: './prices-list.html',
  styleUrl: './prices-list.scss',
})
export class PricesList implements OnInit {
  protected readonly filters: ListingsFilters = { ...DEFAULT_FILTERS };
  protected readonly items = signal<ListingOpportunity[]>([]);
  protected readonly loading = signal(false);
  protected readonly errorMessage = signal('');
  protected readonly total = signal(0);
  protected readonly activeFilters = signal<ListingsFilters>({ ...DEFAULT_FILTERS });
  protected readonly discountSortDirection = signal<'desc' | 'asc'>('desc');

  protected readonly sortedItems = computed(() => {
    const list = [...this.items()];
    const direction = this.discountSortDirection();
    return list.sort((a, b) => {
      const diff = a.discount_percent - b.discount_percent;
      return direction === 'asc' ? diff : -diff;
    });
  });

  protected readonly hasItems = computed(() => this.sortedItems().length > 0);

  protected toggleDiscountSort(): void {
    this.discountSortDirection.update((current) => (current === 'desc' ? 'asc' : 'desc'));
  }

  private readonly api = inject(ListingsApi);

  ngOnInit(): void {
    this.refresh();
  }

  protected refresh(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    const requestFilters = { ...this.filters };
    this.api.getListings(requestFilters).subscribe({
      next: (response) => {
        this.items.set(response.items);
        this.total.set(response.count);
        this.activeFilters.set(response.filters);
        this.loading.set(false);
      },
      error: (error: HttpErrorResponse) => {
        const message =
          (error.error && typeof error.error.error === 'string' && error.error.error) ||
          'No fue posible cargar oportunidades desde el backend.';
        this.items.set([]);
        this.total.set(0);
        this.errorMessage.set(message);
        this.loading.set(false);
      },
    });
  }

  protected getWearInfo(wear: number): { label: string; short: string; badgeClass: string } {
    if (wear < 0.07) {
      return { label: 'Factory New', short: 'FN', badgeClass: 'wear-fn' };
    }
    if (wear < 0.15) {
      return { label: 'Minimal Wear', short: 'MW', badgeClass: 'wear-mw' };
    }
    if (wear < 0.38) {
      return { label: 'Field-Tested', short: 'FT', badgeClass: 'wear-ft' };
    }
    if (wear < 0.45) {
      return { label: 'Well-Worn', short: 'WW', badgeClass: 'wear-ww' };
    }
    return { label: 'Battle-Scarred', short: 'BS', badgeClass: 'wear-bs' };
  }

  protected getFloatPercentage(wear: number): number {
    if (wear == null || Number.isNaN(wear)) return 0;
    return Math.min(Math.max(wear * 100, 0), 100);
  }

  protected parseSkinName(fullName: string): { weapon: string; skin: string } {
    if (!fullName) return { weapon: '', skin: '' };
    const parts = fullName.split('|');
    if (parts.length > 1) {
      const weapon = parts[0].trim();
      const skin = parts.slice(1).join('|').replace(/\s*\([^)]*\)\s*$/, '').trim();
      return { weapon, skin };
    }
    return { weapon: fullName, skin: '' };
  }

  protected calculateProfit(item: ListingOpportunity): number {
    return Math.max(item.steam_reference_price - item.csfloat_price, 0);
  }
}
