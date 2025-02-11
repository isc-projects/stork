import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ZonesPageComponent } from './zones-page.component';

describe('ZonesPageComponent', () => {
  let component: ZonesPageComponent;
  let fixture: ComponentFixture<ZonesPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ZonesPageComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(ZonesPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
