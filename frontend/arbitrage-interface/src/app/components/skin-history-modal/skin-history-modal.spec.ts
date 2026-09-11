import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { ListingsApi } from '../../data/listings-api';
import { SkinHistoryResponse } from '../../types/listings';
import { SkinHistoryModal } from './skin-history-modal';

describe('SkinHistoryModal', () => {
  let fixture: ComponentFixture<SkinHistoryModal>;
  let component: SkinHistoryModal;
  let api: { getSkinHistory: ReturnType<typeof vi.fn> };

  const mockHistory: SkinHistoryResponse = {
    market_hash_name: 'AK-47 | Slate (Field-Tested)',
    graph: [
      { day: '2026-08-01T00:00:00Z', avg_price: 350, avg_price_usd: 3.5, count: 12 },
      { day: '2026-09-01T00:00:00Z', avg_price: 380, avg_price_usd: 3.8, count: 15 },
    ],
    sales: [
      {
        id: 's-1',
        price: 370,
        price_usd: 3.7,
        sold_at: '2026-09-10T15:00:00Z',
        wear: 0.18,
        icon_url: 'https://example.com/icon.png',
      },
    ],
  };

  beforeEach(async () => {
    api = {
      getSkinHistory: vi.fn().mockReturnValue(of(mockHistory)),
    };

    await TestBed.configureTestingModule({
      imports: [SkinHistoryModal],
      providers: [{ provide: ListingsApi, useValue: api }],
    }).compileComponents();

    fixture = TestBed.createComponent(SkinHistoryModal);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('skinName', 'AK-47 | Slate (Field-Tested)');
    fixture.componentRef.setInput('isOpen', true);
  });

  it('loads history data when opened with a skin name', () => {
    fixture.detectChanges();

    expect(api.getSkinHistory).toHaveBeenCalledWith('AK-47 | Slate (Field-Tested)');
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('AK-47 | Slate (Field-Tested)');
    expect(compiled.textContent).toContain('Gráfica de Ventas');
    expect(compiled.textContent).toContain('Últimas Ventas');
  });

  it('renders time range selector buttons', () => {
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const buttons = compiled.querySelectorAll('.range-btn');
    expect(buttons.length).toBe(4);
    expect(buttons[0].textContent).toContain('1M');
    expect(buttons[1].textContent).toContain('3M');
    expect(buttons[2].textContent).toContain('1Y');
    expect(buttons[3].textContent).toContain('ALL');
  });

  it('switches to latest sales tab and renders sales list', () => {
    fixture.detectChanges();

    const salesTabBtn = fixture.nativeElement.querySelectorAll('.tab-btn')[1] as HTMLButtonElement;
    salesTabBtn.click();
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('$3.70');
    expect(compiled.textContent).toContain('0.1800');
  });

  it('emits close when clicking the close button', () => {
    fixture.detectChanges();

    const closeSpy = vi.fn();
    component.close.subscribe(closeSpy);

    const closeBtn = fixture.nativeElement.querySelector('.modal-close') as HTMLButtonElement;
    closeBtn.click();

    expect(closeSpy).toHaveBeenCalled();
  });
});
