import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HaStatusPanelComponent } from './ha-status-panel.component'

import { of } from 'rxjs'

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
        component.serverStatus = of({ state: 'unavailable' })
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
