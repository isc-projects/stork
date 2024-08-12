import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SharedNetworksTableComponent } from './shared-networks-table.component';

describe('SharedNetworksTableComponent', () => {
  let component: SharedNetworksTableComponent;
  let fixture: ComponentFixture<SharedNetworksTableComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SharedNetworksTableComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(SharedNetworksTableComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
