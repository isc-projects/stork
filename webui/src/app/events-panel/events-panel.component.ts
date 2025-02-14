import { Component, OnInit, OnChanges, Input, OnDestroy } from '@angular/core'

import { LazyLoadEvent, MessageService } from 'primeng/api'

import { EventsService, UsersService, ServicesService } from '../backend/api/api'
import { AuthService } from '../auth.service'
import { Subscription, filter, lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Events } from '../backend'
import { ServerSentEventsService } from '../server-sent-events.service'

/**
 * A component that presents the events list. Each event has its own row.
 * The event's text is rendered by EventTextComponent.
 */
@Component({
    selector: 'app-events-panel',
    templateUrl: './events-panel.component.html',
    styleUrls: ['./events-panel.component.sass'],
})
export class EventsPanelComponent implements OnInit, OnChanges, OnDestroy {
    /**
     * A subscription to the events.
     */
    eventSubscription = new Subscription()

    events: Events = { items: [], total: 0 }
    errorCnt = 0
    start = 0
    limit = 10
    loading = false

    /**
     * Contains the IDs of expanded events. Used only in the bare layout.
     */
    expandedEvents = new Set<number>()

    @Input() ui: 'bare' | 'table' = 'bare'

    @Input() filter = {
        level: 0,
        machine: null,
        appType: null,
        daemonType: null,
        user: null,
    }

    /**
     * When set to true, rowsPerPageOptions will be displayed in the paginator.
     */
    @Input() showRowsPerPage = true

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
        public auth: AuthService,
        private sse: ServerSentEventsService
    ) {}

    ngOnDestroy(): void {
        this.eventSubscription.unsubscribe()
    }

    /**
     * Applies new filtering rules.
     *
     * This function is called from ngOnInit or ngOnChanges to apply
     * new filtering rules. It fetches events from the server and
     * re-establishes the SSE connection to receive new events from
     * the server.
     */
    private applyFilter(): void {
        const loadEvent: LazyLoadEvent = { first: 0, rows: this.limit }
        this.refreshEvents(loadEvent)
        this.eventSubscription.unsubscribe()
        this.eventSubscription = this.sse
            .receivePriorityAndMessageEvents(this.filter)
            .pipe(filter((event) => event.stream === 'message'))
            .subscribe((event) => {
                this.eventHandler(event.originalEvent)
            })

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
     * users is only fetched when the logged-in user is a super admin.
     */
    ngOnInit(): void {
        // Indicate that the component was initialized and future calls
        // to ngOnChanges can refresh the events.
        this._initialized = true

        this.applyFilter()

        if (this.isBare) {
            // Bare layout doesn't support data filtration
            this.users = []
            this.machines = []
            return
        }

        if (this.auth.superAdmin()) {
            lastValueFrom(this.usersApi.getUsers(0, 1000, null))
                .then((data) => {
                    this.users = data.items

                    if (this.filter.user) {
                        for (const u of this.users) {
                            if (u.id === this.filter.user) {
                                this.selectedUser = u
                            }
                        }
                    }
                })
                .catch((err) => {
                    const msg = getErrorMessage(err)
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Loading user accounts failed',
                        detail: 'Loading user accounts from the database failed: ' + msg,
                        life: 10000,
                    })
                })
        }
        lastValueFrom(this.servicesApi.getMachines(0, 1000, null, null))
            .then((data) => {
                this.machines = data.items

                if (this.filter.machine) {
                    for (const m of this.machines) {
                        if (m.id === this.filter.machine) {
                            this.selectedMachine = m
                        }
                    }
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get machines',
                    detail: 'Failed to get machines: ' + msg,
                    life: 10000,
                })
            })
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

        this.loading = true

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
            .toPromise()
            .then((data) => {
                this.events.items = data.items ?? []
                this.events.total = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get events',
                    detail: 'Error getting events: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.loading = false
            })
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
            createdAt: event.CreatedAt,
        }

        // put new event in front of all events
        this.events.items.unshift(ev)

        // remove last event if there is too many events
        if (this.events.items.length > this.limit) {
            this.events.items.pop()
        }
        this.events.total += 1
    }

    /** Callback called on selecting a machine in dropdown. */
    onMachineSelect(event) {
        if (event.value === null) {
            this.filter.machine = null
        } else {
            this.filter.machine = event.value.id
        }
        this.applyFilter()
    }

    /** Callback called on selecting an application in dropdown. */
    onAppTypeSelect(event) {
        if (event.value === null) {
            this.filter.appType = null
        } else {
            this.filter.appType = event.value.value
        }
        this.applyFilter()
    }

    /** Callback called on selecting a daemon type in dropdown. */
    onDaemonTypeSelect(event) {
        if (event.value === null) {
            this.filter.daemonType = null
        } else {
            this.filter.daemonType = event.value.value
        }
        this.applyFilter()
    }

    /** Callback called on selecting a user in dropdown. */
    onUserSelect(event) {
        if (event.value === null) {
            this.filter.user = null
        } else {
            this.filter.user = event.value.id
        }
        this.applyFilter()
    }

    /**
     * Toggle the event details expansion. Used only in the bare layout.
     * @param eventId Event ID
     */
    onToggleExpandEventDetails(eventId: number) {
        if (!this.expandedEvents.delete(eventId)) {
            this.expandedEvents.add(eventId)
        }
    }

    /**
     * Returns true if the bare layout is enabled.
     */
    get isBare(): boolean {
        return this.ui === 'bare'
    }

    /**
     * Returns true if the table layout is enabled.
     */
    get isTable(): boolean {
        return this.ui === 'table'
    }
}
