import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DaemonFilterComponent } from './daemon-filter.component'

describe('DaemonFilterComponent', () => {
    let component: DaemonFilterComponent
    let fixture: ComponentFixture<DaemonFilterComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [DaemonFilterComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(DaemonFilterComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
