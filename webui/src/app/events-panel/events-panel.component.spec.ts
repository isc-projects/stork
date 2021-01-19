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

    it('should create event source with correct URL', () => {
        component.filter.level = 1
        component.filter.machine = 2
        component.filter.appType = 'kea'
        component.filter.daemonType = 'dhcp4'
        component.filter.user = 3

        // Event source should be created.
        const source = component.registerServerSentEvents()
        expect(source).toBeTruthy()

        // Capture source's URL.
        const url = new URL(source.url)
        expect(url.pathname).toBe('/sse')

        // Validate parameters.
        const params = url.searchParams
        expect(params.get('level')).toBe('1')
        expect(params.get('machine')).toBe('2')
        expect(params.get('appType')).toBe('kea')
        expect(params.get('daemonName')).toBe('dhcp4')
        expect(params.get('user')).toBe('3')
    })

    it('should refresh events when changes are detected', () => {
        // Capture calls to refreshEvents.
        spyOn(component, 'refreshEvents')
        component.ngOnChanges()
        // ngOnChanges should call refreshEvents function to update the
        // list of events according to new filters.
        expect(component.refreshEvents).toHaveBeenCalled()
    })
})
