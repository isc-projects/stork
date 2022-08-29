import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page.component';

describe('ConfigCheckerPreferencePageComponent', () => {
  let component: ConfigCheckerPreferencePageComponent;
  let fixture: ComponentFixture<ConfigCheckerPreferencePageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ ConfigCheckerPreferencePageComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ConfigCheckerPreferencePageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
