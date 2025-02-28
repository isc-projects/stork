import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ConfigMigrationPageComponent } from './config-migration-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('ConfigMigrationPageComponent', () => {
    let component: ConfigMigrationPageComponent
    let fixture: ComponentFixture<ConfigMigrationPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [ConfigMigrationPageComponent],
            providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()],
        }).compileComponents()

        fixture = TestBed.createComponent(ConfigMigrationPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
