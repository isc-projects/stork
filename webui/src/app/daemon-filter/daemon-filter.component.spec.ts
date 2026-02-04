import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DaemonFilterComponent } from './daemon-filter.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('DaemonFilterComponent', () => {
    let component: DaemonFilterComponent
    let fixture: ComponentFixture<DaemonFilterComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [DaemonFilterComponent],
            providers: [provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(DaemonFilterComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
