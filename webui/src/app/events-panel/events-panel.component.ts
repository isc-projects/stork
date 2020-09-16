import { Component, OnInit, Input } from '@angular/core'

import { MessageService } from 'primeng/api'

import { EventsService, UsersService, ServicesService } from '../backend/api/api'

/**
 * A component that presents events list. Each event has its own row.
 * Event's text is rendered by EventTextComponent.
 */
@Component({
    selector: 'app-events-panel',
    templateUrl: './events-panel.component.html',
    styleUrls: ['./events-panel.component.sass'],
})
export class EventsPanelComponent implements OnInit {
    events: any = { items: [], total: 0 }
    errorCnt = 0
    start = 0
    limit = 10

    @Input() ui = 'bare'

    @Input() filter = {
        level: 0,
        machine: null,
        app: null,
        daemon: null,
        user: null,
    }

    levels = [
        {
            label: 'All',
            value: 0,
            icon: 'pi pi-info-circle',
        },
        {
            label: 'Warnings',
            value: 1,
            icon: 'pi pi-exclamation-triangle',
        },
        {
            label: 'Errors',
            value: 2,
            icon: 'pi pi-exclamation-circle',
        },
    ]

    users: any
    machines: any
    apps: any
    daemons: any
    selectedUser: any
    selectedMachine: any
    selectedApp: any
    selectedDaemon: any

    constructor(
        private eventsApi: EventsService,
        private usersApi: UsersService,
        private servicesApi: ServicesService,
        private msgSrv: MessageService
    ) {}

    ngOnInit(): void {
        this.refreshEvents(null)
        this.registerServerSentEvents()

        this.usersApi.getUsers(0, 1000, null).subscribe(
            (data) => {
                this.users = data.items
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Loading user accounts failed',
                    detail: 'Loading user accounts from the database failed: ' + msg,
                    life: 10000,
                })
            }
        )
        this.servicesApi.getMachines(0, 1000, null, null).subscribe(
            (data) => {
                this.machines = data.items

                if (this.filter.machine) {
                    for (const m of this.machines) {
                        if (m.id === this.filter.machine) {
                            this.selectedMachine = m
                            this.apps = m.apps

                            if (this.filter.app) {
                                for (const a of m.apps) {
                                    if (a.id === this.filter.app) {
                                        this.selectedApp = a
                                        this.daemons = a.details.daemons

                                        if (this.filter.daemon) {
                                            for (const d of a.details.daemons) {
                                                if (d.id === this.filter.daemon) {
                                                    this.selectedDaemon = d
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get machines',
                    detail: 'Getting machines failed: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Load the most recent events from Stork server
     */
    refreshEvents(event) {
        if (event) {
            this.start = event.first
            this.limit = event.rows
        }

        this.eventsApi
            .getEvents(
                this.start,
                this.limit,
                this.filter.level,
                this.filter.daemon,
                this.filter.machine,
                this.filter.user
            )
            .subscribe(
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

    /**
     * Register for SSE for all events.
     */
    registerServerSentEvents() {
        const source = new EventSource('/sse')

        source.addEventListener(
            'error',
            (ev) => {
                // some error appeared - close session and start again but after 10s or 5mins
                console.info('sse error', ev)
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
                console.info('sse data', data)
                this.eventHandler(data)
                // when events are coming then reset error counter
                this.errorCnt = 0
            },
            false
        )

        return source
    }

    /**
     * Take an event received via SSE and put it to list of all events
     * so it is presented in events panel.
     */
    eventHandler(event) {
        // if currently presented page of events is not the first one
        // then do not add new events to the list
        if (this.start !== 0) {
            return
        }
        // decapitalize fields
        const ev = {
            text: event.Text,
            details: event.Details,
            level: event.Level,
            createAt: event.CreateAt,
        }

        // put new event in front of all events
        this.events.items.unshift(ev)

        // remove last event if there is too many events
        if (this.events.items.length > this.limit) {
            this.events.items.pop()
        }
        this.events.total += 1
    }

    expandEvent(ev) {
        if (ev.showDetails === undefined) {
            ev.details = ev.details.replace(/\n/g, '<br>')
        }
        if (ev.showDetails) {
            ev.showDetails = false
        } else {
            ev.showDetails = true
        }
    }

    paginate(event) {
        this.start = event.first
        this.limit = event.rows
        this.refreshEvents(null)
    }

    onMachineSelect(event) {
        if (event.value === null) {
            this.apps = []
            this.filter.machine = null
        } else {
            this.apps = event.value.apps
            this.filter.machine = event.value.id
        }
        this.daemons = []
        this.filter.app = null
        this.filter.daemon = null
        this.refreshEvents(null)
    }

    onAppSelect(event) {
        console.info(event)
        if (event.value === null) {
            this.daemons = []
            this.filter.app = null
        } else {
            this.daemons = event.value.details.daemons
            this.filter.app = event.value.id
        }
        this.filter.daemon = null
        this.refreshEvents(null)
    }

    onDaemonSelect(event) {
        console.info(event)
        if (event.value) {
            this.filter.daemon = event.value.id
        } else {
            this.filter.daemon = null
        }
        this.refreshEvents(null)
    }
}
