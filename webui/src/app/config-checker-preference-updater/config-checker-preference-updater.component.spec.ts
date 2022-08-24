import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater.component';

describe('ConfigCheckerPreferenceUpdaterComponent', () => {
  let component: ConfigCheckerPreferenceUpdaterComponent;
  let fixture: ComponentFixture<ConfigCheckerPreferenceUpdaterComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ ConfigCheckerPreferenceUpdaterComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ConfigCheckerPreferenceUpdaterComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
