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
  protected readonly hasItems = computed(() => this.items().length > 0);

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
}
