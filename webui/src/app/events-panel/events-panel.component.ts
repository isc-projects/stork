import { Component, OnInit } from '@angular/core'

import { MessageService } from 'primeng/api'

import { EventsService } from '../backend/api/api'

@Component({
    selector: 'app-events-panel',
    templateUrl: './events-panel.component.html',
    styleUrls: ['./events-panel.component.sass'],
})
export class EventsPanelComponent implements OnInit {
    events: any = []
    errorCnt = 0

    constructor(private eventsApi: EventsService, private msgSrv: MessageService) {}

    ngOnInit(): void {
        this.refreshEvents()
        this.registerServerSentEvents()
    }

    refreshEvents() {
        this.eventsApi.getEvents().subscribe(
            (data) => {
                this.events = data
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get events',
                    detail: 'Getting events erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    registerServerSentEvents() {
        const source = new EventSource('/sse')

        source.addEventListener(
            'error',
            (ev) => {
                // some error appeared - close session and start again but after 10s or 5mins
                source.close()
                this.errorCnt += 1
                if (this.errorCnt < 10) {
                    // try to re-register every 10s but only 10 times
                    setTimeout(() => {
                        this.registerServerSentEvents()
                    }, 10000)
                } else {
                    // try to re-register every 5mins if there are too many errors
                    setTimeout(() => {
                        this.registerServerSentEvents()
                    }, 600000)
                }
            },
            false
        )

        source.addEventListener(
            'message',
            (ev) => {
                const data = JSON.parse(ev.data)
                this.eventHandler(data)
                // when events are coming then reset error counter
                this.errorCnt = 0
            },
            false
        )

        return source
    }

    eventHandler(event) {
        // decapitalize fields
        const ev = {
            text: event.Text,
            leve: event.Level,
        }

        // put new event in front of all events
        this.events.items.unshift(ev)

        // remove last event if there is too many events
        if (this.events.items.length > 30) {
            this.events.items.pop()
        }
        this.events.total += 1
    }
}
