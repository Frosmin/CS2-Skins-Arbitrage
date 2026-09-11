import { CommonModule, DecimalPipe, DatePipe } from '@angular/common';
import {
  Component,
  ElementRef,
  OnDestroy,
  ViewChild,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { Chart, registerables } from 'chart.js';

import { ListingsApi } from '../../data/listings-api';
import {
  HistoryGraphPoint,
  RecentSaleRecord,
  SkinHistoryResponse,
} from '../../types/listings';

Chart.register(...registerables);

@Component({
  selector: 'app-skin-history-modal',
  standalone: true,
  imports: [CommonModule, DecimalPipe, DatePipe],
  templateUrl: './skin-history-modal.html',
  styleUrl: './skin-history-modal.scss',
})
export class SkinHistoryModal implements OnDestroy {
  readonly skinName = input.required<string>();
  readonly iconUrl = input<string>('');
  readonly currentPrice = input<number>(0);
  readonly isOpen = input<boolean>(false);

  readonly close = output<void>();

  @ViewChild('chartCanvas') chartCanvas?: ElementRef<HTMLCanvasElement>;

  protected readonly activeTab = signal<'graph' | 'sales'>('graph');
  protected readonly activeRange = signal<'1M' | '3M' | '1Y' | 'ALL'>('1Y');
  protected readonly loading = signal(false);
  protected readonly errorMessage = signal('');
  protected readonly historyData = signal<SkinHistoryResponse | null>(null);

  private chartInstance: Chart | null = null;
  private readonly api = inject(ListingsApi);

  constructor() {
    effect(() => {
      const open = this.isOpen();
      const name = this.skinName();
      if (open && name) {
        this.loadHistory(name);
      } else {
        this.destroyChart();
      }
    });

    effect(() => {
      const tab = this.activeTab();
      const range = this.activeRange();
      const data = this.historyData();
      if (tab === 'graph' && data && this.isOpen()) {
        setTimeout(() => this.renderChart(data.graph, range), 0);
      }
    });
  }

  ngOnDestroy(): void {
    this.destroyChart();
  }

  protected loadHistory(name: string): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.api.getSkinHistory(name).subscribe({
      next: (data) => {
        this.historyData.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        this.errorMessage.set(
          (err.error && typeof err.error.error === 'string' && err.error.error) ||
            'Error al cargar historial de ventas.',
        );
        this.loading.set(false);
      },
    });
  }

  protected setTab(tab: 'graph' | 'sales'): void {
    this.activeTab.set(tab);
  }

  protected setRange(range: '1M' | '3M' | '1Y' | 'ALL'): void {
    this.activeRange.set(range);
  }

  protected handleBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }

  private renderChart(rawPoints: HistoryGraphPoint[], range: '1M' | '3M' | '1Y' | 'ALL'): void {
    if (!this.chartCanvas) return;

    this.destroyChart();

    const filtered = this.filterPointsByRange(rawPoints, range);
    if (filtered.length === 0) return;

    const labels = filtered.map((pt) => {
      const d = new Date(pt.day);
      return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: '2-digit' });
    });
    const priceData = filtered.map((pt) => pt.avg_price_usd);
    const volumeData = filtered.map((pt) => pt.count);

    const ctx = this.chartCanvas.nativeElement.getContext('2d');
    if (!ctx) return;

    const gradient = ctx.createLinearGradient(0, 0, 0, 300);
    gradient.addColorStop(0, 'rgba(59, 130, 246, 0.35)');
    gradient.addColorStop(1, 'rgba(59, 130, 246, 0.0)');

    this.chartInstance = new Chart(ctx, {
      type: 'line',
      data: {
        labels,
        datasets: [
          {
            label: 'Precio promedio (USD)',
            data: priceData,
            borderColor: '#3b82f6',
            backgroundColor: gradient,
            borderWidth: 2,
            fill: true,
            tension: 0.2,
            pointRadius: filtered.length > 60 ? 0 : 2,
            pointHoverRadius: 5,
            pointHoverBackgroundColor: '#60a5fa',
            yAxisID: 'y',
          },
          {
            type: 'bar',
            label: 'Volumen vendido',
            data: volumeData,
            backgroundColor: 'rgba(148, 163, 184, 0.25)',
            hoverBackgroundColor: 'rgba(148, 163, 184, 0.5)',
            borderRadius: 2,
            yAxisID: 'y1',
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: {
          mode: 'index',
          intersect: false,
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: '#0f172a',
            titleColor: '#94a3b8',
            bodyColor: '#ffffff',
            borderColor: 'rgba(255, 255, 255, 0.1)',
            borderWidth: 1,
            padding: 10,
            callbacks: {
              label: (context) => {
                if (context.dataset.yAxisID === 'y') {
                  return ` Precio: $${Number(context.parsed.y).toFixed(2)} USD`;
                }
                return ` Vendidos: ${context.parsed.y}`;
              },
            },
          },
        },
        scales: {
          x: {
            grid: { color: 'rgba(255, 255, 255, 0.05)' },
            ticks: { color: '#64748b', maxRotation: 0, autoSkip: true, maxTicksLimit: 7 },
          },
          y: {
            position: 'right',
            grid: { color: 'rgba(255, 255, 255, 0.05)' },
            ticks: {
              color: '#94a3b8',
              callback: (val) => `$${val}`,
            },
          },
          y1: {
            position: 'left',
            display: false,
            grid: { display: false },
            min: 0,
            max: Math.max(...volumeData, 1) * 4,
          },
        },
      },
    });
  }

  private filterPointsByRange(points: HistoryGraphPoint[], range: '1M' | '3M' | '1Y' | 'ALL'): HistoryGraphPoint[] {
    if (range === 'ALL' || points.length === 0) return points;

    const days = range === '1M' ? 30 : range === '3M' ? 90 : 365;
    const cutoff = new Date();
    cutoff.setDate(cutoff.getDate() - days);

    return points.filter((p) => new Date(p.day) >= cutoff);
  }

  private destroyChart(): void {
    if (this.chartInstance) {
      this.chartInstance.destroy();
      this.chartInstance = null;
    }
  }
}
