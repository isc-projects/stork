import { Component, OnInit, OnChanges, Input } from '@angular/core'

import { MessageService } from 'primeng/api'

import { EventsService, UsersService, ServicesService } from '../backend/api/api'
import { AuthService } from '../auth.service'

/**
 * A component that presents events list. Each event has its own row.
 * Event's text is rendered by EventTextComponent.
 */
@Component({
    selector: 'app-events-panel',
    templateUrl: './events-panel.component.html',
    styleUrls: ['./events-panel.component.sass'],
})
export class EventsPanelComponent implements OnInit, OnChanges {
    events: any = { items: [], total: 0 }
    errorCnt = 0
    start = 0
    limit = 10

    @Input() ui = 'bare'

    @Input() filter = {
        level: 0,
        machine: null,
        appType: null,
        daemonType: null,
        user: null,
    }

    levels = [
        {
            label: 'All',
            id: 'all-events',
            value: 0,
            icon: 'pi pi-info-circle',
        },
        {
            label: 'Warnings',
            id: 'warning-events',
            value: 1,
            icon: 'pi pi-exclamation-triangle',
        },
        {
            label: 'Errors',
            id: 'error-events',
            value: 2,
            icon: 'pi pi-exclamation-circle',
        },
    ]

    users: any
    machines: any
    appTypes = [
        { value: 'kea', name: 'Kea', id: 'kea-events' },
        { value: 'bind9', name: 'BIND 9', id: 'bind-events' },
    ]
    daemonTypes = [
        { value: 'dhcp4', name: 'DHCPv4', id: 'kea4-events' },
        { value: 'dhcp6', name: 'DHCPv6', id: 'kea6-events' },
        { value: 'named', name: 'named', id: 'named-events' },
        { value: 'd2', name: 'DDNS', id: 'ddns-events' },
        { value: 'ca', name: 'CA', id: 'ca-events' },
        { value: 'netconf', name: 'NETCONF', id: 'netconf-events' },
    ]
    selectedMachine: any
    selectedAppType: any
    selectedDaemonType: any
    selectedUser: any

    eventSource: EventSource

    /**
     * Indicates if the component was initialized.
     *
     * It is used by ngOnChanges to determine if the events should
     * be refreshed. The ngOnChanges is called before ngOnInit and
     * we should avoid refreshing the events in both calls. If this
     * is the first call to ngOnChanges the events are not refreshed
     * and we let ngOnInit load them. The ngOnInit sets this flag to
     * true. Later, ngOnChanges refreshes the events when the filter
     * changes are detected.
     */
    private _initialized = false

    constructor(
        private eventsApi: EventsService,
        private usersApi: UsersService,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        public auth: AuthService
    ) {}

    /**
     * Applies new filtering rules.
     *
     * This function is called from ngOnInit or ngOnChanges to apply
     * new filtering rules. It fetches events from the server and
     * re-establishes the SSE connection to receive new events from
     * the server.
     */
    private applyFilter(): void {
        this.refreshEvents(null)
        this.registerServerSentEvents()

        if (this.filter.appType) {
            for (const at of this.appTypes) {
                if (at.value === this.filter.appType) {
                    this.selectedAppType = at
                    break
                }
            }
        }
        if (this.filter.daemonType) {
            for (const dt of this.daemonTypes) {
                if (dt.value === this.filter.daemonType) {
                    this.selectedDaemonType = dt
                    break
                }
            }
        }
    }

    /**
     * Component lifecycle hook called to initialize the data.
     *
     * This function fetches the events and establishes the SSE connection
     * to the server using the specified filter. It also fetches the machines
     * and users from the server. The users and machines are used to initialize
     * drop down controls which can be used to modify the filtering rules,
     * e.g. select events pertaining to the particular machine. The list of
     * users is only fetched when the logged in user is a super admin.
     */
    ngOnInit(): void {
        // Indicate that the component was intialized and future calls
        // to ngOnChanges can refresh the events.
        this._initialized = true

        this.applyFilter()

        if (this.auth.superAdmin()) {
            this.usersApi.getUsers(0, 1000, null).subscribe(
                (data) => {
                    this.users = data.items

                    if (this.filter.user) {
                        for (const u of this.users) {
                            if (u.id === this.filter.user) {
                                this.selectedUser = u
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
                        summary: 'Loading user accounts failed',
                        detail: 'Loading user accounts from the database failed: ' + msg,
                        life: 10000,
                    })
                }
            )
        }
        this.servicesApi.getMachines(0, 1000, null, null).subscribe(
            (data) => {
                this.machines = data.items

                if (this.filter.machine) {
                    for (const m of this.machines) {
                        if (m.id === this.filter.machine) {
                            this.selectedMachine = m
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
     * Component lifecycle hook called when data bound to the component change.
     *
     * If this function is called after ngOnInit, it refreshes the events using
     * new filtering rules and re-establishes SSE connection to the server.
     */
    ngOnChanges(): void {
        // Refresh the events only after the component was initialized
        // and the events were loaded by the ngOnInit. If this is the
        // first call to ngOnChanges, don't refresh the events.
        if (this._initialized) {
            this.applyFilter()
        }
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
                this.filter.machine,
                this.filter.appType,
                this.filter.daemonType,
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
     * Establishes the SSE connection to the server.
     *
     * If the connection already exists, it is closed and a new connection
     * is established.
     */
    registerServerSentEvents() {
        // Close existing connection.
        if (this.eventSource) {
            this.eventSource.close()
        }

        // Build event source URL from filters.
        const searchParams = new URLSearchParams()
        if (this.filter.machine) {
            searchParams.append('machine', String(this.filter.machine))
        }
        if (this.filter.appType) {
            searchParams.append('appType', this.filter.appType)
        }
        if (this.filter.daemonType) {
            searchParams.append('daemonName', this.filter.daemonType)
        }
        if (this.filter.user) {
            searchParams.append('user', String(this.filter.user))
        }
        if (this.filter.level) {
            searchParams.append('level', String(this.filter.level))
        }
        this.eventSource = new EventSource('/sse?' + searchParams.toString())

        this.eventSource.addEventListener(
            'error',
            (ev) => {
                // some error appeared - close session and start again but after 10s or 5mins
                console.info('sse error', ev)
                this.eventSource.close()
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

        this.eventSource.addEventListener(
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
            this.filter.machine = null
        } else {
            this.filter.machine = event.value.id
        }
        this.applyFilter()
    }

    onAppTypeSelect(event) {
        if (event.value === null) {
            this.filter.appType = null
        } else {
            this.filter.appType = event.value.value
        }
        this.applyFilter()
    }

    onDaemonTypeSelect(event) {
        if (event.value === null) {
            this.filter.daemonType = null
        } else {
            this.filter.daemonType = event.value.value
        }
        this.applyFilter()
    }

    onUserSelect(event) {
        if (event.value === null) {
            this.filter.user = null
        } else {
            this.filter.user = event.value.id
        }
        this.applyFilter()
    }
}
