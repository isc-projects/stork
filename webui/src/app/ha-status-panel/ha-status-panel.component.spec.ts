import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HaStatusPanelComponent } from './ha-status-panel.component'

describe('HaStatusPanelComponent', () => {
    let component: HaStatusPanelComponent
    let fixture: ComponentFixture<HaStatusPanelComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [HaStatusPanelComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusPanelComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
