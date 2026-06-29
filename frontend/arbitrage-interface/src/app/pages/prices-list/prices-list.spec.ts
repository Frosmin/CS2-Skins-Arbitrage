import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PricesList } from './prices-list';

describe('PricesList', () => {
  let component: PricesList;
  let fixture: ComponentFixture<PricesList>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PricesList]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PricesList);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
