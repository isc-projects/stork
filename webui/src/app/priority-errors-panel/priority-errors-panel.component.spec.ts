import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { PriorityErrorsPanelComponent } from './priority-errors-panel.component'
import { ServicesService } from '../backend'
import { MessageService } from 'primeng/api'
import {
    EventStream,
    SSEEvent,
    ServerSentEventsService,
    ServerSentEventsTestingService,
} from '../server-sent-events.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessagesModule } from 'primeng/messages'
import { Subject, of, throwError } from 'rxjs'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { HttpErrorResponse } from '@angular/common/http'

describe('PriorityErrorsPanelComponent', () => {
    let component: PriorityErrorsPanelComponent
    let fixture: ComponentFixture<PriorityErrorsPanelComponent>
    let messageService: MessageService
    let sse: ServerSentEventsService
    let api: ServicesService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                ServicesService,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
            ],
            imports: [HttpClientTestingModule, MessagesModule, NoopAnimationsModule],
            declarations: [PriorityErrorsPanelComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(PriorityErrorsPanelComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
        messageService = fixture.debugElement.injector.get(MessageService)
        sse = fixture.debugElement.injector.get(ServerSentEventsService)
        api = fixture.debugElement.injector.get(ServicesService)
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should receive events and get connectivity status from the server', fakeAsync(() => {
        // Create a source of events.
        let receivedEventsSubject = new Subject<SSEEvent>()
        // Create an observable that the component subscribes to in order to receive the events.
        let observable = receivedEventsSubject.asObservable()
        spyOn(sse, 'receivePriorityEvents').and.returnValue(observable)

        // Simulate returning an app with the connectivity issues.
        const apps: any = {
            items: [
                {
                    id: 1,
                },
            ],
            total: 1,
        }

        // No unauthorized machines.
        const unauthorized: any = 0

        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(of(unauthorized))
        spyOn(component, 'setBackoff').and.callThrough()
        spyOn(component, 'setBackoffTimeout')

        // When the component is initialized it should subscribe to the events and
        // receive the report about the apps with connectivity issues.
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(1)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)

        expect(component.setBackoff).toHaveBeenCalledTimes(2)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Connectivity, true)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Registration, true)
        expect(component.messages.length).toBe(1)

        // To prevent the storm of requests to the server for each received event
        // we use a backoff mechanism to delay any next request after receiving
        // the status.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)

        // Simulate receiving next event indicating connectivity issues.
        receivedEventsSubject.next({
            stream: 'connectivity',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()

        // The backoff has been enabled so the new event should not trigger
        // any API calls.
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(1)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)

        // The event count should be raised, though.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(1)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)

        expect(component.messages.length).toBe(1)

        // Disable the backoff. Normally it goes away after a timeout on
        // its own.
        component.setBackoff(EventStream.Connectivity, false)
        component.resetEventCount(EventStream.Connectivity)

        // Send another event. This time we should fetch an updated state
        // from the server.
        receivedEventsSubject.next({
            stream: 'connectivity',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(2)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)

        // The backoff should still be enabled.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)

        expect(component.messages.length).toBe(1)
    }))

    it('should receive events and get unauthorized machines count from the server', fakeAsync(() => {
        // Create a source of events.
        let receivedEventsSubject = new Subject<SSEEvent>()
        // Create an observable that the component subscribes to in order to receive the events.
        let observable = receivedEventsSubject.asObservable()
        spyOn(sse, 'receivePriorityEvents').and.returnValue(observable)

        // Simulate no connectivity issues.
        const apps: any = {
            items: [],
            total: 0,
        }

        // First, return no unauthorized machines. Return some in the second call.
        const unauthorized: any[] = [0, 2]

        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValues(of(unauthorized[0]), of(unauthorized[1]))
        spyOn(component, 'setBackoff').and.callThrough()
        spyOn(component, 'setBackoffTimeout')

        // When the component is initialized it should subscribe to the events and
        // receive the report about the apps with connectivity issues.
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(1)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)

        expect(component.setBackoff).toHaveBeenCalledTimes(2)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Connectivity, true)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Registration, true)
        expect(component.setBackoff).toHaveBeenCalledTimes(2)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Connectivity, true)
        expect(component.setBackoff).toHaveBeenCalledWith(EventStream.Registration, true)

        expect(component.messages.length).toBe(0)

        // To prevent the storm of requests to the server for each received event
        // we use a backoff mechanism to delay any next request after receiving
        // the status.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)

        // Simulate receiving an event indicating new registration requests.
        receivedEventsSubject.next({
            stream: 'registration',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()

        // The backoff has been enabled so the new event should not trigger
        // any API calls.
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(1)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)

        // The event count should be raised, though.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(1)

        expect(component.messages.length).toBe(0)

        // Disable the backoff. Normally it goes away after a timeout on
        // its own.
        component.setBackoff(EventStream.Registration, false)
        component.resetEventCount(EventStream.Registration)

        // Send another event. This time we should fetch an updated state
        // from the server.
        receivedEventsSubject.next({
            stream: 'registration',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(1)
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(2)

        // The backoff should still be enabled.
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)

        expect(component.messages.length).toBe(1)
        expect(component.messages[0].key).toBe('registration')
    }))

    it('should display warnings for both connectivity issues and registration requests', fakeAsync(() => {
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        // Simulate returning an app with issues.
        const apps: any = {
            items: [
                {
                    id: 1,
                },
            ],
            total: 1,
        }
        const unauthorized: any = 1
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(of(unauthorized))
        spyOn(component, 'setBackoffTimeout')
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalled()

        expect(component.messages.length).toBe(2)
    }))

    it('should display no issues', fakeAsync(() => {
        // Create a source of events.
        let receivedEventsSubject = new Subject<SSEEvent>()

        // Create an observable that the component subscribes to in order to receive the events.
        let observable = receivedEventsSubject.asObservable()
        spyOn(sse, 'receivePriorityEvents').and.returnValue(observable)

        // Simulate returning no connectivity issues.
        const apps: any = {
            items: [],
            total: 0,
        }
        // Also, no unauthorized machines.
        const unauthorized: any = 0
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(of(unauthorized))
        spyOn(component, 'setBackoffTimeout')

        // When the component is initialized it should subscribe to the events and
        // receive the report about the apps with connectivity issues and unauthorized
        // machines.
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalled()
        expect(component.setBackoffTimeout).toHaveBeenCalled()

        expect(component.messages.length).toBe(0)
        expect(component.isBackoff(EventStream.Connectivity)).toBeTrue()
        expect(component.getEventCount(EventStream.Connectivity)).toBe(0)
        expect(component.isBackoff(EventStream.Registration)).toBeTrue()
        expect(component.getEventCount(EventStream.Registration)).toBe(0)
    }))

    it('should unsubscribe when the component is destroyed', fakeAsync(() => {
        const apps: any = {
            items: [],
            total: 0,
        }
        const unauthorized: any = 0
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(of(unauthorized))
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        spyOn(component, 'setBackoffTimeout')
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalled()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()

        spyOn(component.subscription, 'unsubscribe')
        component.ngOnDestroy()
        expect(component.subscription.unsubscribe).toHaveBeenCalled()
    }))

    it('should display an error message while getting connectivity issues', fakeAsync(() => {
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        // Simulate an error while fetching the apps.
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )
        const unauthorized: any = 0
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(of(unauthorized))
        spyOn(component, 'setBackoffTimeout')
        spyOn(messageService, 'add')
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
        expect(component.messages.length).toBe(0)
    }))

    it('should display an error message while getting unauthorized machines', fakeAsync(() => {
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        // Return empty list of apps with the connectivity issues.
        const apps: any = {
            items: [],
            total: 0,
        }
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(apps)

        // Simulate returning an error while getting unauthorized machines.
        spyOn(api, 'getUnauthorizedMachinesCount').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )
        spyOn(component, 'setBackoffTimeout')
        spyOn(messageService, 'add')
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(api.getUnauthorizedMachinesCount).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
        expect(component.messages.length).toBe(0)
    }))
})
