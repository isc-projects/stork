import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { ActivatedRoute, Router } from '@angular/router'
import { MessageService } from 'primeng/api'

import { EventsService, ServicesService, UsersService } from '../backend'
import { EventsPanelComponent } from './events-panel.component'

/**
 * Fake event value.
 */
class TestEventValue {
    public id: any
    public value: any
}

/**
 * Fake event object.
 */
class TestEvent {
    public value: TestEventValue
    constructor() {
        this.value = new TestEventValue()
    }
}

describe('EventsPanelComponent', () => {
    let component: EventsPanelComponent
    let fixture: ComponentFixture<EventsPanelComponent>

    function itContainsSearchParam(name, value) {
        const source = component.eventSource
        expect(source).toBeTruthy()

        // Capture source's URL.
        const url = new URL(source.url)
        expect(url.pathname).toBe('/sse')

        // Make sure the source is using correct filtering.
        const params = url.searchParams
        expect(params.get(name)).toBe(value)
    }

    beforeEach(
        waitForAsync(() => {
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
        })
    )

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
        component.registerServerSentEvents()
        const source = component.eventSource
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

    it('should update event source after changes', () => {
        // Set initial filter.
        component.filter.level = 1

        // Create event source using this filter.
        component.registerServerSentEvents()
        let source = component.eventSource
        expect(source).toBeTruthy()

        // Capture source's URL.
        const url = new URL(source.url)
        expect(url.pathname).toBe('/sse')

        // Make sure the source is using correct filtering.
        itContainsSearchParam('level', '1')

        // Change the filter.
        component.filter.level = 2

        // Calling this again should cause the old SSE connection to
        // be closed and create new connection using the new filter.
        component.registerServerSentEvents()
        source = component.eventSource
        expect(source).toBeTruthy()

        // Make sure the filter was applied correctly.
        itContainsSearchParam('level', '2')
    })

    it('should refresh events when changes are detected', () => {
        // Capture calls to refreshEvents and registerServerSentEvents.
        spyOn(component, 'refreshEvents')
        spyOn(component, 'registerServerSentEvents')
        component.ngOnChanges()
        // ngOnChanges should call refreshEvents function to update the
        // list of events according to new filters. It should also call
        // the registerServerSentEvents to subscribe to the updates.
        expect(component.refreshEvents).toHaveBeenCalled()
        expect(component.registerServerSentEvents).toHaveBeenCalled()
    })

    it('should re-establish SSE connection on events', () => {
        component.registerServerSentEvents()

        const event = new TestEvent()

        // Select specific machine, app type, daemon type and user. In each
        // case, the SSE connection should be re-established with appropriate
        // filtering parameters.

        event.value.id = 1
        component.onMachineSelect(event)
        itContainsSearchParam('machine', '1')

        event.value.value = 'kea'
        component.onAppTypeSelect(event)
        itContainsSearchParam('appType', 'kea')

        event.value.value = 'dhcp4'
        component.onDaemonTypeSelect(event)
        itContainsSearchParam('daemonName', 'dhcp4')

        event.value.id = 5
        component.onUserSelect(event)
        itContainsSearchParam('user', '5')
    })
})
