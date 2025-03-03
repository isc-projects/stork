import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ConfigMigrationTabComponent } from './config-migration-tab.component'

describe('ConfigMigrationTabComponent', () => {
    let component: ConfigMigrationTabComponent
    let fixture: ComponentFixture<ConfigMigrationTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [ConfigMigrationTabComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(ConfigMigrationTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
