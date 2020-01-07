import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HaStatusComponent } from './ha-status.component'

describe('HaStatusComponent', () => {
    let component: HaStatusComponent
    let fixture: ComponentFixture<HaStatusComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [HaStatusComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
