import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { EventTextComponent } from './event-text.component'

describe('EventTextComponent', () => {
    let component: EventTextComponent
    let fixture: ComponentFixture<EventTextComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [EventTextComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventTextComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
