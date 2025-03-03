import { ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { MenuItem, MessageService } from 'primeng/api'
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
     * Collection of Zone Inventory States fetched from backend that are presented in Zones Fetching Status table.
     */
    zonesFetchingStates: ZoneInventoryState[] = []

    /**
     * Total count of Zone Inventory States fetched from backend.
     */
    zonesFetchingStatesTotal: number = 0

    /**
     * Flag stating whether Zones Fetching Status table data is loading or not.
     */
    zonesFetchingStatesLoading: boolean = false

    /**
     * Timestamp formatting used to display "Created at" or "Loaded at" data.
     */
    dateTimeFormat = 'YYYY-MM-dd HH:mm:ss'

    /**
     * Flag stating whether Zones fetching is in progress or not.
     */
    fetchingInProgress: boolean = false

    /**
     * Keeps count of DNS apps for which Zones fetching was completed. This number comes from backend.
     */
    fetchingAppsCompletedCount: number = 0

    /**
     * Keeps total count of DNS apps for which Zones fetching is currently in progress. This number comes from backend.
     */
    fetchingTotalAppsCount: number = 0

    /**
     * Collection of open tabs with zones details.
     */
    openTabs: Zone[] = []

    /**
     * Keeps active zone details tab index.
     */
    activeIdx: number = 0

    /**
     * Flag stating whether Zones Fetching Status dialog is visible or not.
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
     * RxJS observable which locks Fetch Zones button for 5 seconds to limit the rate of PUT Zones Fetch requests sent.
     * @private
     */
    private _putZonesFetchGuard = of(null).pipe(
        tap(() => (this.putZonesFetchLocked = true)),
        concatMap(() => timer(5000)),
        tap(() => (this.putZonesFetchLocked = false)),
        share()
    )

    /**
     * Key to be used in browser storage for keeping Zone Fetch Sent flag value.
     * @private
     */
    private _fetchSentStorageKey = 'zone-fetch-sent'

    /**
     * Interval in milliseconds between requests sent to backend REST API asking about Zones fetching status.
     * @private
     */
    private _pollingInterval: number = 10 * 1000

    /**
     * RxJS observable which emits one value with DNS Service GET ZonesFetch request response together with the HTTP response
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
     * RxJS observable stream which sends GET ZonesFetch request response every interval of _pollingInterval time
     * until Zone fetching is complete OR fetchingInProgress is set to false. It is useful for polling the Zone fetching status
     * once 202 Accepted response is received after GET ZonesFetch request.
     * Expected sequence of sent values is: 202 ZonesFetchStatus -> ... -> 202 ZonesFetchStatus -> 200 ZoneInventoryStates |-> complete.
     * @private
     */
    private _polling$ = interval(this._pollingInterval).pipe(
        switchMap(() => this._getZonesFetchWithStatus), // Use switchMap to discard ongoing request from previous interval tick.
        takeWhile((resp) => this.fetchingInProgress && resp.status === HttpStatusCode.Accepted, true),
        tap((resp) => {
            if (resp.status === HttpStatusCode.Accepted) {
                this.fetchingAppsCompletedCount = resp.completedAppsCount
                this.fetchingTotalAppsCount = resp.appsCount
            } else if (resp.status === HttpStatusCode.Ok) {
                this.fetchingAppsCompletedCount = this.fetchingTotalAppsCount
                this.zonesFetchingStates = resp.items ?? []
                this.zonesFetchingStatesTotal = resp.total ?? 0
                if (this.fetchingInProgress) {
                    this.messageService.add({
                        severity: 'success',
                        summary: 'Zone Fetching done',
                        detail: 'Zone Fetching finished successfully!',
                        life: 5000,
                    })
                    if (this.zonesFetchingStates.length > 0) {
                        this.zoneInventoryStateMap = new Map()
                        this.zonesFetchingStates.forEach((s) => {
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
                detail: 'Sending GET Zones Fetch request failed: ' + msg,
                life: 10000,
            })
            return of(EMPTY) // In case of any GET ZonesFetch error, just display Error feedback in UI and complete this observable.
        }),
        finalize(() => {
            this.fetchingInProgress = false
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
     */
    constructor(
        private cd: ChangeDetectorRef,
        private dnsService: DNSService,
        private messageService: MessageService
    ) {}

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        // console.log('onInit')
        // TODO: should it be called onInit?
        this.refreshFetchingStatusTable()
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
        if (event.index <= this.activeIdx) {
            this.activeIdx = 0
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
            // tab exists, just switch to it
            this.activeIdx = zoneIdx + 1
        } else {
            this.openTabs = [...this.openTabs, zone]
            this.cd.detectChanges()
            console.log('openTab setting activeIdx to', this.openTabs.length)
            this.activeIdx = this.openTabs.length
        }
    }

    /**
     * Fetches data from backend and refreshes Zones Fetching Status table with the data.
     * If Zone fetching is in progress, it subscribes to _polling$ observable to receive and
     * visualize fetching progress.
     */
    refreshFetchingStatusTable() {
        this.zonesFetchingStatesLoading = true
        lastValueFrom(this._getZonesFetchWithStatus)
            .then((resp) => {
                switch (resp.status) {
                    case HttpStatusCode.NoContent:
                        this.fetchingInProgress = false
                        this.messageService.add({
                            severity: 'info',
                            summary: 'No Zone Fetching information',
                            detail: 'Information about Zone Fetching is currently unavailable.',
                            life: 5000,
                        })
                        break
                    case HttpStatusCode.Accepted:
                        this.fetchingInProgress = true
                        this.zonesFetchingStates = []
                        this.zonesFetchingStatesTotal = 0

                        this.fetchingAppsCompletedCount = resp.completedAppsCount
                        this.fetchingTotalAppsCount = resp.appsCount

                        if (!this._isPolling) {
                            this._isPolling = true
                            this._subscriptions.add(this._polling$.subscribe())
                        }

                        break
                    case HttpStatusCode.Ok:
                        this.zonesFetchingStates = resp.items ?? []
                        this.zonesFetchingStatesTotal = resp.total ?? 0
                        if (this.zonesFetchingStates.length > 0) {
                            this.zoneInventoryStateMap = new Map()
                            this.zonesFetchingStates.forEach((s) => {
                                this.zoneInventoryStateMap.set(s.daemonId, s)
                            })
                        }

                        if (this.fetchingInProgress) {
                            this.fetchingInProgress = false
                            this.fetchingAppsCompletedCount = this.fetchingTotalAppsCount
                            this.messageService.add({
                                severity: 'success',
                                summary: 'Zone Fetching done',
                                detail: 'Zone Fetching finished successfully!',
                                life: 5000,
                            })
                            this.onLazyLoadZones(this.zonesTable?.createLazyLoadMetadata())
                        }

                        break
                    default:
                        this.fetchingInProgress = false
                        this.messageService.add({
                            severity: 'info',
                            summary: 'Unexpected response',
                            detail:
                                'Unexpected response to GET Zones Fetch request - received HTTP status code ' +
                                resp.status,
                            life: 5000,
                        })
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Sending GET Zones Fetch request failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.zonesFetchingStatesLoading = false
            })
    }

    /**
     * Sends PUT ZonesFetch request and triggers refreshing data of the Zones Fetching Status table right after.
     */
    sendPutZonesFetch() {
        this._subscriptions.add(this._putZonesFetchGuard.subscribe())

        lastValueFrom(
            this.dnsService.putZonesFetch().pipe(
                tap(() => (this.fetchingInProgress = true)),
                delay(500), // Trigger refreshFetchingStatusTable() with small delay - smaller deployments will likely have 200 Ok ZoneInventoryStates response there.
                concatMap((resp) => {
                    this.refreshFetchingStatusTable()
                    return of(resp)
                })
            )
        )
            .then(() => {
                this.storeZoneFetchSent(true)
                this.messageService.add({
                    severity: 'success',
                    summary: 'Request sent',
                    detail: 'Sending PUT Zones Fetch request succeeded.',
                    life: 5000,
                })
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Sending PUT Zones Fetch request failed: ' + msg,
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
                return 'Zone inventory on an agent performs a long lasting operation and cannot perform the requested operation at this time.'
            case 'ok':
                return 'Communication with the zone inventory was successful.'
            case 'erred':
                return 'Error when communicating with a zone inventory on an agent.'
            case 'uninitialized':
                return 'Zone inventory was not initialized (neither populated nor loaded).'
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
        this.cd.detectChanges()
        lastValueFrom(this.dnsService.getZones(event?.first ?? 0, event?.rows ?? 10))
            .then((resp) => {
                this.zonesExpandedRows = {}
                this.zones = resp?.items ?? []
                this.zonesTotal = resp?.total ?? 0

                if (this.zones.length === 0 && !this.wasZoneFetchSent()) {
                    this.sendPutZonesFetch()
                    this.messageService.add({
                        severity: 'info',
                        summary: 'Automatically fetching zones',
                        detail: 'Zones were not fetched yet, so Fetch Zones was triggered automatically.',
                        life: 5000,
                    })
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error retrieving data',
                    detail: 'Retrieving Zones data failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => (this.zonesLoading = false))
    }

    /**
     * Retrieves information from browser session storage whether PUT Zones Fetch was sent or not.
     */
    wasZoneFetchSent(): boolean {
        const fromStorage = sessionStorage.getItem(this._fetchSentStorageKey) ?? 'false'
        return JSON.parse(fromStorage) === true
    }

    /**
     * Stores information in browser session storage whether PUT Zones Fetch was sent or not.
     * @param sent PUT Zones Fetch was sent or not
     */
    storeZoneFetchSent(sent: boolean) {
        sessionStorage.setItem(this._fetchSentStorageKey, JSON.stringify(sent))
    }
}
