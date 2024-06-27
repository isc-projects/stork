import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { PaginatorModule } from 'primeng/paginator'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'

import { EventsService, ServicesService, UsersService } from '../backend'
import { EventTextComponent } from '../event-text/event-text.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { EventsPanelComponent } from './events-panel.component'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { of } from 'rxjs'

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
    let sseService: ServerSentEventsService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                EventsService,
                UsersService,
                ServicesService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
            ],
            imports: [
                HttpClientTestingModule,
                PaginatorModule,
                RouterTestingModule,
                TableModule,
                ToastModule,
                ButtonModule,
            ],
            declarations: [EventsPanelComponent, LocaltimePipe, EventTextComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPanelComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
        sseService = fixture.debugElement.injector.get(ServerSentEventsService)
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should establish SSE connection with correct filtering rules', () => {
        component.filter.level = 1
        component.filter.machine = 2
        component.filter.appType = 'kea'
        component.filter.daemonType = 'dhcp4'
        component.filter.user = 3

        spyOn(sseService, 'receivePriorityAndMessageEvents').and.returnValue(
            of({
                stream: 'foo',
                originalEvent: {},
            })
        )

        component.ngOnInit()
        fixture.detectChanges()

        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledOnceWith(component.filter)
    })

    it('should renew subscription upon filter changes', () => {
        component.filter.level = 1

        spyOn(sseService, 'receivePriorityAndMessageEvents').and.returnValue(
            of({
                stream: 'foo',
                originalEvent: {},
            })
        )

        component.ngOnInit()
        fixture.detectChanges()

        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledTimes(1)
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)

        component.filter.level = 2
        component.ngOnChanges()

        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledTimes(2)
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)
    })

    it('should re-establish SSE connection on events', () => {
        spyOn(sseService, 'receivePriorityAndMessageEvents').and.returnValue(
            of({
                stream: 'foo',
                originalEvent: {},
            })
        )

        component.ngOnInit()
        fixture.detectChanges()

        // Select specific machine, app type, daemon type and user. In each
        // case, the SSE connection should be re-established with appropriate
        // filtering parameters.

        const event = new TestEvent()

        event.value.id = 1
        component.onMachineSelect(event)
        expect(component.filter.machine).toBe(1)
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)

        event.value.value = 'kea'
        component.onAppTypeSelect(event)
        expect(component.filter.appType).toBe('kea')
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)

        event.value.value = 'dhcp4'
        component.onDaemonTypeSelect(event)
        expect(component.filter.daemonType).toBe('dhcp4')
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)

        event.value.id = 5
        component.onUserSelect(event)
        expect(component.filter.user).toBe(5)
        expect(sseService.receivePriorityAndMessageEvents).toHaveBeenCalledWith(component.filter)
    })

    it('should unsubscribe from events on destroy', () => {
        spyOn(sseService, 'receivePriorityAndMessageEvents').and.returnValue(
            of({
                stream: 'foo',
                originalEvent: {},
            })
        )
        component.ngOnInit()
        fixture.detectChanges()

        spyOn(component.eventSubscription, 'unsubscribe')
        component.ngOnDestroy()
        expect(component.eventSubscription.unsubscribe).toHaveBeenCalled()
    })

    it('should recognize the layout type', () => {
        component.ui = 'table'
        expect(component.isBare).toBeFalse()
        expect(component.isTable).toBeTrue()
        component.ui = 'bare'
        expect(component.isBare).toBeTrue()
        expect(component.isTable).toBeFalse()
    })

    it('should toggle event details expansion', () => {
        expect(component.expandedEvents.size).toBe(0)

        component.onToggleExpandEventDetails(42)
        expect(component.expandedEvents.has(42)).toBeTrue()

        component.onToggleExpandEventDetails(42)
        expect(component.expandedEvents.has(42)).toBeFalse()
    })
})
