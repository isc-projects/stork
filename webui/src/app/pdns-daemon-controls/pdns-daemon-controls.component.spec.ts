import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PdnsDaemonControlsComponent } from './pdns-daemon-controls.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideRouter } from '@angular/router'

describe('PdnsDaemonControlsComponent', () => {
    let component: PdnsDaemonControlsComponent
    let fixture: ComponentFixture<PdnsDaemonControlsComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [provideHttpClientTesting(), provideHttpClient(withInterceptorsFromDi()), provideRouter([])],
        }).compileComponents()

        fixture = TestBed.createComponent(PdnsDaemonControlsComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
