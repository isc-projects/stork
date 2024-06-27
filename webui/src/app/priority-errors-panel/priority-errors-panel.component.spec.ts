import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { PriorityErrorsPanelComponent } from './priority-errors-panel.component'
import { ServicesService } from '../backend'
import { MessageService } from 'primeng/api'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessagesModule } from 'primeng/messages'
import { BehaviorSubject, of, throwError } from 'rxjs'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

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
        let receivedEventsSubject = new BehaviorSubject({
            stream: 'all',
            originalEvent: null,
        })
        // Create an observable the component subscribes to to receive the events.
        let observable = receivedEventsSubject.asObservable()
        spyOn(sse, 'receivePriorityEvents').and.returnValue(observable)
        // Simulate returning an app with the connectivity issues.
        let apps: any = {
            items: [
                {
                    id: 1,
                },
            ],
            total: 1,
        }
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(component, 'setBackoffTimeout')

        // When the component is initialized it should subscibe to the events and
        // receive the report about the apps with connectivity issues.
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(component.setBackoffTimeout).toHaveBeenCalled()
        expect(component.messages.length).toBe(1)
        // To prevent the storm of requests to the server for each received event
        // we use a backoff mechanism to delay any next request after receiving
        // the status.
        expect(component.backoff).toBeTrue()
        expect(component.eventCount).toBe(0)

        // Simulate receiving next event.
        receivedEventsSubject.next({
            stream: 'connectivity',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()

        // It should cause the component to fetch the apps again.
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(2)
        expect(component.messages.length).toBe(1)
        expect(component.backoff).toBeTrue()
        expect(component.eventCount).toBe(1)

        // Simulate receiving another event.
        receivedEventsSubject.next({
            stream: 'connectivity',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()

        // It should not trigger any new subscriptions nor requests to the
        // server because we have the backoff enabled.
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(2)
        expect(component.messages.length).toBe(1)
        expect(component.backoff).toBeTrue()
        expect(component.eventCount).toBe(2)

        // Disable the backoff. Normally it goes away after a timeout on
        // its own.
        component.backoff = false
        component.eventCount = 0

        // Send another event. This time we should fetch an updated state
        // from the server.
        receivedEventsSubject.next({
            stream: 'connectivity',
            originalEvent: null,
        })
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalledTimes(1)
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalledTimes(3)
        expect(component.messages.length).toBe(1)
        expect(component.backoff).toBeTrue()
        expect(component.eventCount).toBe(0)
    }))

    it('should not display any warnings when the number of apps is 0', fakeAsync(() => {
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        // Simulate returning an empty list of apps.
        let apps: any = {
            items: [],
            total: 0,
        }
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(component, 'setBackoffTimeout')
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(component.messages.length).toBe(0)
    }))

    it('should unsubscribe when the component is detroyed', fakeAsync(() => {
        let apps: any = {
            items: [],
            total: 0,
        }
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(of(apps))
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        spyOn(component, 'setBackoffTimeout')
        component.ngOnInit()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(sse.receivePriorityEvents).not.toHaveBeenCalled()
        tick()
        spyOn(component.subscription, 'unsubscribe')
        component.ngOnDestroy()
        expect(component.subscription.unsubscribe).toHaveBeenCalled()
    }))

    it('should display an error message', fakeAsync(() => {
        spyOn(sse, 'receivePriorityEvents').and.returnValue(
            of({
                stream: 'all',
                originalEvent: null,
            })
        )
        // Simulate an error while fetching the apps.
        spyOn(api, 'getAppsWithCommunicationIssues').and.returnValue(throwError({ status: 404 }))
        spyOn(component, 'setBackoffTimeout')
        spyOn(messageService, 'add')
        component.ngOnInit()
        fixture.detectChanges()
        tick()
        expect(sse.receivePriorityEvents).toHaveBeenCalled()
        expect(api.getAppsWithCommunicationIssues).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
        expect(component.messages.length).toBe(0)
    }))
})
