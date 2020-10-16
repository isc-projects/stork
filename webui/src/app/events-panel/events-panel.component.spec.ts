import { HttpClientTestingModule } from '@angular/common/http/testing'
import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { ActivatedRoute, Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { EventsService, ServicesService, UsersService } from '../backend'

import { EventsPanelComponent } from './events-panel.component'

describe('EventsPanelComponent', () => {
    let component: EventsPanelComponent
    let fixture: ComponentFixture<EventsPanelComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                EventsService,
                UsersService,
                ServicesService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
            ],
            imports: [HttpClientTestingModule],
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
