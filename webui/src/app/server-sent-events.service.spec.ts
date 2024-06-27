import { TestBed } from '@angular/core/testing'

import { ServerSentEventsService } from './server-sent-events.service'

describe('ServerSentEventsService', () => {
    let service: ServerSentEventsService

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(ServerSentEventsService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should receive the connectivity events', () => {
        spyOn(service, 'addEventListeners')

        let observable = service.receivePriorityEvents()
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith('/sse?stream=connectivity&stream=registration')
    })

    it('should receive the connectivity and message events', () => {
        spyOn(service, 'addEventListeners')

        let filter = {}
        let observable = service.receivePriorityAndMessageEvents(filter)
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?stream=connectivity&stream=registration&stream=message'
        )
    })

    it('should receive the message events with filters', () => {
        spyOn(service, 'addEventListeners')

        let filter = {
            level: 1,
            machine: 2,
            appType: 'kea',
            daemonType: 'dhcp4',
            user: 'foo',
        }
        let observable = service.receivePriorityAndMessageEvents(filter)
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=kea&daemonName=dhcp4&user=foo&level=1&stream=connectivity&stream=registration&stream=message'
        )
    })

    it('should not reconnect when the connectivity events are subscribed to', () => {
        spyOn(service, 'addEventListeners')

        let filter = {}
        let observable = service.receivePriorityAndMessageEvents(filter)
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?stream=connectivity&stream=registration&stream=message'
        )

        observable = service.receivePriorityEvents()
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledOnceWith(
            '/sse?stream=connectivity&stream=registration&stream=message'
        )
    })

    it('should reconnect when the message events are not subscribed to', () => {
        spyOn(service, 'addEventListeners')
        spyOn(service, 'closeEventSource')

        let observable = service.receivePriorityEvents()
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith('/sse?stream=connectivity&stream=registration')
        expect(service.closeEventSource).toHaveBeenCalledTimes(1)

        let filter = {}
        observable = service.receivePriorityAndMessageEvents(filter)
        expect(observable).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(2)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?stream=connectivity&stream=registration&stream=message'
        )
        expect(service.closeEventSource).toHaveBeenCalledTimes(2)
    })

    it('should reconnect when the filtering rules change', () => {
        spyOn(service, 'addEventListeners')

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 1,
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=1&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(2)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'kea',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(3)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=kea&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(4)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=bind9&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
                daemonType: 'bind9',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(5)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=bind9&daemonName=bind9&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
                daemonType: 'foo',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(6)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=bind9&daemonName=foo&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
                daemonType: 'foo',
                user: 'bar',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(7)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=bind9&daemonName=foo&user=bar&stream=connectivity&stream=registration&stream=message'
        )

        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
                daemonType: 'foo',
                user: 'abc',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(8)
        expect(service.addEventListeners).toHaveBeenCalledWith(
            '/sse?machine=2&appType=bind9&daemonName=foo&user=abc&stream=connectivity&stream=registration&stream=message'
        )

        // If the filtering rules don't change there should be no attempt to reconnect.
        expect(
            service.receivePriorityAndMessageEvents({
                machine: 2,
                appType: 'bind9',
                daemonType: 'foo',
                user: 'abc',
            })
        ).toBeTruthy()
        expect(service.addEventListeners).toHaveBeenCalledTimes(8)
    })
})
