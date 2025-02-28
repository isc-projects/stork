import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfigMigrationPageComponent } from './config-migration-page.component';

describe('ConfigMigrationPageComponent', () => {
  let component: ConfigMigrationPageComponent;
  let fixture: ComponentFixture<ConfigMigrationPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfigMigrationPageComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(ConfigMigrationPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
