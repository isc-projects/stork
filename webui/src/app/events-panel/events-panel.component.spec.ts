import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { EventsPanelComponent } from './events-panel.component'

describe('EventsPanelComponent', () => {
    let component: EventsPanelComponent
    let fixture: ComponentFixture<EventsPanelComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [EventsPanelComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPanelComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
