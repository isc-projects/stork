import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfigMigrationTableComponent } from './config-migration-table.component';

describe('ConfigMigrationTableComponent', () => {
  let component: ConfigMigrationTableComponent;
  let fixture: ComponentFixture<ConfigMigrationTableComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfigMigrationTableComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(ConfigMigrationTableComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
