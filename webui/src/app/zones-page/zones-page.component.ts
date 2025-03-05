import { ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ConfirmationService, MenuItem, MessageService } from 'primeng/api'
import { Zone, DNSService, ZonesFetchStatus, ZoneInventoryStates, ZoneInventoryState } from '../backend'
import { TabViewCloseEvent } from 'primeng/tabview'
import { concatMap, finalize, share, switchMap, takeWhile, tap, delay, catchError, map } from 'rxjs/operators'
import { EMPTY, interval, lastValueFrom, of, Subscription, timer } from 'rxjs'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { getErrorMessage } from '../utils'
import { HttpResponse, HttpStatusCode } from '@angular/common/http'

/**
 * Type defining Status of the Zone Inventory returned from the backend.
 */
type ZoneInventoryStatus = 'busy' | 'erred' | 'ok' | 'uninitialized' | string

@Component({
    selector: 'app-zones-page',
    templateUrl: './zones-page.component.html',
    styleUrl: './zones-page.component.sass',
})
export class ZonesPageComponent implements OnInit, OnDestroy {
    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs: MenuItem[] = [{ label: 'DNS' }, { label: 'Zones' }]

    /**
     * Collection of zones fetched from backend.
     */
    zones: Zone[] = []

    /**
     * Total count of zones fetched from backend.
     */
    zonesTotal: number = 0

    /**
     * Flag stating whether zones table data is loading or not.
     */
    zonesLoading: boolean = false

    /**
     * Unique identifier of a stateful zones table to be used in browser storage.
     */
    zonesStateKey: string = 'zones-table-state'

    /**
     * Keeps expanded rows of zones table.
     */
    zonesExpandedRows = {}

    /**
     * PrimeNG table component containing list of all zones.
     */
    @ViewChild('zonesTable') zonesTable: Table

    /**
     * Collection of Zone Inventory States fetched from backend that are presented in Zones Fetch Status table.
     */
    zonesFetchStates: ZoneInventoryState[] = []

    /**
     * Total count of Zone Inventory States fetched from backend.
     */
    zonesFetchStatesTotal: number = 0

    /**
     * Flag stating whether Zones Fetch Status table data is loading or not.
     */
    zonesFetchStatesLoading: boolean = false

    /**
     * Flag stating whether zones fetch is in progress or not.
     */
    fetchInProgress: boolean = false

    /**
     * Keeps count of DNS apps for which zones fetch was completed. This number comes from backend.
     */
    fetchAppsCompletedCount: number = 0

    /**
     * Keeps total count of DNS apps for which zones fetch is currently in progress. This number comes from backend.
     */
    fetchTotalAppsCount: number = 0

    /**
     * Collection of open tabs with zones details.
     */
    openTabs: Zone[] = []

    /**
     * Keeps active zone details tab index.
     */
    activeTabIdx: number = 0

    /**
     * Flag stating whether Zones Fetch Status dialog is visible or not.
     */
    fetchStatusVisible: boolean = false

    /**
     * Map keeping Zone Inventory States accessible by the daemonId key.
     */
    zoneInventoryStateMap: Map<number, Partial<ZoneInventoryState>> = new Map()

    /**
     * Flag stating whether Fetch Zones button is locked/disabled or not.
     */
    putZonesFetchLocked: boolean = false

    /**
     * Column names for tables which display local zones.
     */
    localZoneColumns: string[] = ['App Name', 'App ID', 'View', 'Zone Type', 'Serial', 'Class', 'Loaded At']

    /**
     * Reference to Array ctor to be used in the HTML template.
     * @protected
     */
    protected readonly Array = Array

    /**
     * RxJS observable which locks Fetch Zones button for 5 seconds to limit the rate of PUT /dns-management/zones-fetch requests sent.
     * @private
     */
    private _putZonesFetchGuard = of(null).pipe(
        tap(() => (this.putZonesFetchLocked = true)),
        concatMap(() => timer(5000)),
        tap(() => (this.putZonesFetchLocked = false)),
        share()
    )

    /**
     * Key to be used in browser storage for keeping zones fetch sent flag value.
     * @private
     */
    private _fetchSentStorageKey = 'zone-fetch-sent'

    /**
     * Interval in milliseconds between requests sent to backend REST API asking about zones fetch status.
     * @private
     */
    private _pollingInterval: number = 10 * 1000

    /**
     * RxJS observable which emits one value with the GET /dns-management/zones-fetch response
     * status header value and completes.
     * @private
     */
    private _getZonesFetchWithStatus = this.dnsService.getZonesFetch('response').pipe(
        map((httpResponse: HttpResponse<ZonesFetchStatus & ZoneInventoryStates>) => ({
            ...httpResponse.body,
            status: httpResponse.status,
        }))
    )

    /**
     * RxJS observable stream which returns a response to GET /dns-management/zones-fetch request every interval of _pollingInterval time
     * until zones fetch is complete OR fetchInProgress is set to false. It is useful for polling the zones fetch status
     * once 202 Accepted response is received after GET /dns-management/zones-fetch request.
     * Expected sequence of sent values is: 202 ZonesFetchStatus -> ... -> 202 ZonesFetchStatus -> 200 ZoneInventoryStates |-> complete.
     * @private
     */
    private _polling$ = interval(this._pollingInterval).pipe(
        switchMap(() => this._getZonesFetchWithStatus), // Use switchMap to discard ongoing request from previous interval tick.
        takeWhile((resp) => this.fetchInProgress && resp.status === HttpStatusCode.Accepted, true),
        tap((resp) => {
            if (resp.status === HttpStatusCode.Accepted) {
                this.fetchAppsCompletedCount = resp.completedAppsCount
                this.fetchTotalAppsCount = resp.appsCount
            } else if (resp.status === HttpStatusCode.Ok) {
                this.fetchAppsCompletedCount = this.fetchTotalAppsCount
                this.zonesFetchStates = resp.items ?? []
                this.zonesFetchStatesTotal = resp.total ?? 0
                if (this.fetchInProgress) {
                    this.messageService.add({
                        severity: 'success',
                        summary: 'Zones fetch complete',
                        detail: 'Zones fetched successfully!',
                        life: 5000,
                    })
                    if (this.zonesFetchStates.length > 0) {
                        this.zoneInventoryStateMap = new Map()
                        this.zonesFetchStates.forEach((s) => {
                            this.zoneInventoryStateMap.set(s.daemonId, s)
                        })
                    }

                    this.onLazyLoadZones(this.zonesTable?.createLazyLoadMetadata())
                }
            }
        }),
        share(), // This should not happen, but in case there is more than one subscriber of this observable, use share to have only one interval running and sharing results to all subscribers.
        catchError((err) => {
            const msg = getErrorMessage(err)
            this.messageService.add({
                severity: 'error',
                summary: 'Error sending request',
                detail: 'Sending GET /dns-management/zones-fetch request failed: ' + msg,
                life: 10000,
            })
            return of(EMPTY) // In case of any GET /dns-management/zones-fetch error, just display Error feedback in UI and complete this observable.
        }),
        finalize(() => {
            this.fetchInProgress = false
            this._isPolling = false
        })
    )

    /**
     * RxJS subscriptions kept in one place to unsubsribe all at once when this component is destroyed.
     * @private
     */
    private _subscriptions = new Subscription()

    /**
     * Flag stating whether _polling$ observable is active (emitting values and not complete yet) or not.
     * @private
     */
    private _isPolling = false

    /**
     * Class constructor.
     * @param cd Angular change detection required to manually trigger detectChanges in this component
     * @param dnsService service providing DNS REST APIs
     * @param messageService PrimeNG message service used to display feedback messages in UI
     * @param confirmationService PrimeNG confirmation service used to display confirmation dialog
     */
    constructor(
        private cd: ChangeDetectorRef,
        private dnsService: DNSService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService
    ) {}

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        this.refreshFetchStatusTable()
    }

    /**
     * Component lifecycle hook which destroys the component.
     */
    ngOnDestroy() {
        this._subscriptions.unsubscribe()
    }

    /**
     * Callback called when detailed zone tab is closed.
     * @param event closing event
     */
    onTabClose(event: TabViewCloseEvent) {
        this.openTabs.splice(event.index - 1, 1)
        if (event.index <= this.activeTabIdx) {
            this.activeTabIdx = 0
        }
    }

    /**
     * Opens tab with zone details.
     * @param zone zone to be displayed in details
     */
    openTab(zone: Zone) {
        const openTabsZoneIds = this.openTabs.map((z) => z.id)
        const zoneIdx = openTabsZoneIds.indexOf(zone.id)
        if (zoneIdx >= 0) {
            // Tab exists, just switch to it.
            this.activeTabIdx = zoneIdx + 1
        } else {
            this.openTabs = [...this.openTabs, zone]
            this.cd.detectChanges()
            this.activeTabIdx = this.openTabs.length
        }
    }

    /**
     * Fetches data from backend and refreshes Zones Fetch Status table with the data.
     * If zones fetch is in progress, it subscribes to _polling$ observable to receive and
     * visualize Fetch progress.
     */
    refreshFetchStatusTable() {
        this.zonesFetchStatesLoading = true
        lastValueFrom(this._getZonesFetchWithStatus)
            .then((resp) => {
                switch (resp.status) {
                    case HttpStatusCode.NoContent:
                        this.fetchInProgress = false
                        this.zonesFetchStates = []
                        this.zonesFetchStatesTotal = 0
                        this.messageService.add({
                            severity: 'info',
                            summary: 'Zones not fetched',
                            detail: 'Zones fetch status not available because zones have not been fetched yet.',
                            life: 5000,
                        })
                        break
                    case HttpStatusCode.Accepted:
                        this.fetchInProgress = true
                        this.zonesFetchStates = []
                        this.zonesFetchStatesTotal = 0

                        this.fetchAppsCompletedCount = resp.completedAppsCount
                        this.fetchTotalAppsCount = resp.appsCount

                        if (!this._isPolling) {
                            this._isPolling = true
                            this._subscriptions.add(this._polling$.subscribe())
                        }

                        break
                    case HttpStatusCode.Ok:
                        this.zonesFetchStates = resp.items ?? []
                        this.zonesFetchStatesTotal = resp.total ?? 0
                        if (this.zonesFetchStates.length > 0) {
                            this.zoneInventoryStateMap = new Map()
                            this.zonesFetchStates.forEach((s) => {
                                this.zoneInventoryStateMap.set(s.daemonId, s)
                            })
                        }

                        if (this.fetchInProgress) {
                            this.fetchInProgress = false
                            this.fetchAppsCompletedCount = this.fetchTotalAppsCount
                            this.messageService.add({
                                severity: 'success',
                                summary: 'Zones fetch complete',
                                detail: 'Zones fetched successfully!',
                                life: 5000,
                            })
                            this.onLazyLoadZones(this.zonesTable?.createLazyLoadMetadata())
                        }

                        break
                    default:
                        this.fetchInProgress = false
                        this.messageService.add({
                            severity: 'info',
                            summary: 'Unexpected response',
                            detail:
                                'Unexpected response while fetching zones - received HTTP status code ' + resp.status,
                            life: 5000,
                        })
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Fetching zones failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.zonesFetchStatesLoading = false
            })
    }

    /**
     * Wrapper function that calls _sendPutZonesFetch() but only after user confirms that action
     * in the confirmation dialog.
     * @param autoAccept when set to true, the confirmation step will be skipped; defaults to false
     */
    sendPutZonesFetch(autoAccept: boolean = false) {
        if (autoAccept) {
            this._sendPutZonesFetch()
        } else {
            this.confirmationService.confirm({
                message:
                    'This operation instructs the Stork server to fetch the zones from all DNS servers' +
                    ' and update them in the Stork database. This operation may take long time depending' +
                    ' on the number of the DNS servers and zones. All zones data will be overwritten with the newly ' +
                    'fetched data from DNS servers that Stork is monitoring. Are you sure you want to continue?',
                header: 'Confirm Fetching Zones!',
                icon: 'pi pi-exclamation-triangle',
                acceptLabel: 'Continue',
                rejectLabel: 'Cancel',
                accept: () => {
                    this._sendPutZonesFetch()
                },
            })
        }
    }

    /**
     * Sends PUT /dns-management/zones-fetch request and triggers refreshing data of the Zones Fetch Status table right after.
     * @private
     */
    private _sendPutZonesFetch() {
        this._subscriptions.add(this._putZonesFetchGuard.subscribe())

        lastValueFrom(
            this.dnsService.putZonesFetch().pipe(
                tap(() => (this.fetchInProgress = true)),
                delay(500), // Trigger refreshFetchStatusTable() with small delay - smaller deployments will likely have 200 Ok ZoneInventoryStates response there.
                concatMap((resp) => {
                    this.refreshFetchStatusTable()
                    return of(resp)
                })
            )
        )
            .then(() => {
                this.storeZoneFetchSent(true)
                this.messageService.add({
                    severity: 'success',
                    summary: 'Request sent',
                    detail: 'Fetching zones started successfully.',
                    life: 5000,
                })
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Fetching zones failed: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Returns PrimeNG severity for given ZoneInventoryStatus.
     * @param status ZoneInventoryStatus
     */
    getSeverity(status: ZoneInventoryStatus) {
        switch (status) {
            case 'ok':
                return 'success'
            case 'busy':
                return 'warning'
            case 'erred':
                return 'danger'
            case 'uninitialized':
                return 'secondary'
            default:
                return 'info'
        }
    }

    /**
     * Returns more verbose error message for given error.
     * @param err error message received from backend
     */
    getStateErrorMessage(err: string) {
        return `Error when communicating with a zone inventory on an agent: ${err}.`
    }

    /**
     * Returns tooltip message for given ZoneInventoryStatus.
     * @param status ZoneInventoryStatus
     */
    getTooltip(status: ZoneInventoryStatus) {
        switch (status) {
            case 'busy':
                return 'Zone inventory on the agent is busy and cannot return zones at this time. Try again later.'
            case 'ok':
                return 'Stork server successfully fetched all zones from the DNS server.'
            case 'erred':
                return 'Error when communicating with a zone inventory on an agent.'
            case 'uninitialized':
                return 'Zone inventory on the agent was not initialized. Trying again or restarting the agent can help.'
            default:
                return null
        }
    }

    /**
     * Lazily loads paged zones data from backend.
     * @param event PrimeNG TableLazyLoadEvent with metadata about table pagination.
     */
    onLazyLoadZones(event: TableLazyLoadEvent) {
        this.zonesLoading = true
        this.cd.detectChanges() // in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        lastValueFrom(this.dnsService.getZones(event?.first ?? 0, event?.rows ?? 10))
            .then((resp) => {
                this.zonesExpandedRows = {}
                this.zones = resp?.items ?? []
                this.zonesTotal = resp?.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error retrieving zones',
                    detail: 'Retrieving zones information failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => (this.zonesLoading = false))
    }

    /**
     * Retrieves information from browser session storage whether PUT /dns-management/zones-fetch was sent or not.
     */
    wasZoneFetchSent(): boolean {
        const fromStorage = sessionStorage.getItem(this._fetchSentStorageKey) ?? 'false'
        return JSON.parse(fromStorage) === true
    }

    /**
     * Stores information in browser session storage whether PUT /dns-management/zones-fetch was sent or not.
     * @param sent request was sent or not
     */
    storeZoneFetchSent(sent: boolean) {
        sessionStorage.setItem(this._fetchSentStorageKey, JSON.stringify(sent))
    }
}
