import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { ActivatedRoute } from '@angular/router'

import { EventsPageComponent } from './events-page.component'
import { EventsService } from '../backend/api/events.service'

describe('EventsPageComponent', () => {
    let component: EventsPageComponent
    let fixture: ComponentFixture<EventsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [ EventsService, {
                provide: ActivatedRoute,
                useValue: {
                    snapshot: { queryParams: {} },
                },
            }
        ],
            declarations: [EventsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
