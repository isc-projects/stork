import { ChangeDetectorRef, Component, NgZone, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ConfirmationService, MenuItem, MessageService, TableState } from 'primeng/api'
import {
    DNSAppType,
    DNSClass,
    DNSService,
    DNSZoneType,
    LocalZone,
    Zone,
    ZoneInventoryState,
    ZoneInventoryStates,
    ZonesFetchStatus,
} from '../backend'
import {
    catchError,
    concatMap,
    delay,
    distinctUntilChanged,
    finalize,
    map,
    share,
    switchMap,
    takeWhile,
    tap,
} from 'rxjs/operators'
import { debounceTime, EMPTY, interval, lastValueFrom, of, Subject, Subscription, timer } from 'rxjs'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { getErrorMessage, unrootZone } from '../utils'
import { HttpResponse, HttpStatusCode } from '@angular/common/http'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { Router } from '@angular/router'
import { getTooltip, getSeverity } from '../zone-inventory-utils'

/**
 * An interface extending the LocalZone with the properties useful
 * in the component template.
 */
interface ExtendedLocalZone extends LocalZone {
    disableShowZone: boolean
}

@Component({
    selector: 'app-zones-page',
    standalone: false,
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
     * Keeps number of zones per page in the zones table.
     */
    zonesRows: number = 10

    /**
     * Key to be used in browser storage for keeping zones table state.
     * @private
     */
    private readonly _zonesTableStateStorageKey = 'zones-table-state'

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
     * Flag stating whether Zones Fetch Status dialog is visible or not.
     */
    fetchStatusVisible: boolean = false

    /**
     * Map keeping Zone Inventory States accessible by the daemonId key.
     */
    zoneInventoryStateMap: Map<number, ZoneInventoryState> = new Map()

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
     * Returns RxJS observable which emits one value with the GET /dns-management/zones-fetch response
     * together with the status header value and completes.
     * @return RxJS observable
     */
    getZonesFetchWithStatus() {
        return this.dnsService.getZonesFetch('response').pipe(
            map((httpResponse: HttpResponse<ZonesFetchStatus & ZoneInventoryStates>) => ({
                ...httpResponse.body,
                status: httpResponse.status,
            }))
        )
    }

    /**
     * RxJS observable stream which returns a response to GET /dns-management/zones-fetch request every interval of _pollingInterval time
     * until zones fetch is complete OR fetchInProgress is set to false. It is useful for polling the zones fetch status
     * once 202 Accepted response is received after GET /dns-management/zones-fetch request.
     * Expected sequence of sent values is: 202 ZonesFetchStatus -> ... -> 202 ZonesFetchStatus -> 200 ZoneInventoryStates |-> complete.
     * @private
     */
    private _polling$ = interval(this._pollingInterval).pipe(
        switchMap(() => this.getZonesFetchWithStatus()), // Use switchMap to discard ongoing request from previous interval tick.
        takeWhile((resp) => this.fetchInProgress && resp.status === HttpStatusCode.Accepted, true),
        tap((resp) => {
            if (resp.status === HttpStatusCode.Accepted) {
                this.fetchAppsCompletedCount = resp.completedAppsCount
                this.fetchTotalAppsCount = resp.appsCount
                this.onLazyLoadZones(this.zonesTable?.createLazyLoadMetadata(), false)
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
                    this._resetZoneInventoryStateMap()
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
     * RxJS subscriptions kept in one place to unsubscribe all at once when this component is destroyed.
     * @private
     */
    private _subscriptions: Subscription

    /**
     * Flag stating whether _polling$ observable is active (emitting values and not complete yet) or not.
     * @private
     */
    private _isPolling: boolean = false

    /**
     * RxJS Subject used for filtering zones table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _zonesTableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * Map keeping zone viewer dialog visibility state for each zone viewer dialog.
     */
    zoneViewerDialogVisible: Map<string, boolean> = new Map()

    /**
     * Class constructor.
     * @param cd Angular change detection required to manually trigger detectChanges in this component
     * @param dnsService service providing DNS REST APIs
     * @param messageService PrimeNG message service used to display feedback messages in UI
     * @param confirmationService PrimeNG confirmation service used to display confirmation dialog
     * @param router Angular router service used to navigate when zones table filtering changes
     * @param zone Angular zone to call Router navigation inside the zone
     */
    constructor(
        private cd: ChangeDetectorRef,
        private dnsService: DNSService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService,
        private router: Router,
        private zone: NgZone
    ) {}

    /**
     * Zone types values used for the UI filter dropdown options.
     */
    zoneTypes: string[] = []

    /**
     * Zone classes values used for the UI filter dropdown options.
     */
    zoneClasses: string[] = []

    /**
     * Response policy zone filtering options.
     *
     * The options are:
     * - include - include RPZ with other zones,
     * - exclude - exclude RPZ from the list,
     * - only - return only RPZ.
     */
    rpzOptions: { name: string; value: string }[] = [
        { name: 'include', value: 'include' },
        { name: 'exclude', value: 'exclude' },
        { name: 'only', value: 'only' },
    ]

    /**
     * DNS app types values used for the UI filter dropdown options.
     */
    appTypes: { name: string; value: string }[] = []

    /**
     * Returns label for the DNS App type.
     * @param appType DNS App type
     * @return App type label
     */
    private _getDNSAppName(appType: DNSAppType) {
        switch (appType) {
            case DNSAppType.Bind9:
                return 'BIND9'
            case DNSAppType.Pdns:
                return 'PowerDNS'
            default:
                return (<string>appType).toUpperCase()
        }
    }

    /**
     * Object containing supported zone filters which values are provided via URL deep-link.
     * The properties of this object correspond to queryParam keys.
     * Values of this object describe:
     * - filter type (numeric, enum, string or boolean)
     * - filter matchMode (contains, equals) which corresponds to PrimeNG table filter metadata
     * - accepted enum values for enum type of filters
     * - array type; when set to true it means that the filter may use more than one value.
     */
    supportedQueryParamFilters: {
        [k: string]: {
            type: 'numeric' | 'enum' | 'string' | 'boolean'
            matchMode: 'contains' | 'equals'
            enumValues?: string[]
            arrayType?: boolean
        }
    }

    /**
     * Restores only rows per page count for the zones table from the state stored in user browser storage.
     * @private
     */
    private _restoreZonesTableRowsPerPage() {
        const stateString = localStorage.getItem(this._zonesTableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.zonesRows = state.rows ?? 10
        }
    }

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        this.supportedQueryParamFilters = {
            appId: { type: 'numeric', matchMode: 'contains' },
            appType: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSAppType) },
            zoneType: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSZoneType), arrayType: true },
            rpz: { type: 'enum', matchMode: 'equals', enumValues: ['include', 'exclude', 'only'] },
            zoneClass: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSClass) },
            text: { type: 'string', matchMode: 'contains' },
            zoneSerial: { type: 'string', matchMode: 'contains' },
        }

        // Initialize arrays that contain values for UI filter dropdowns.
        for (const t in DNSZoneType) {
            this.zoneTypes.push(DNSZoneType[t])
        }

        for (const c in DNSClass) {
            if (DNSClass[c] === DNSClass.Any) {
                continue
            }

            this.zoneClasses.push(DNSClass[c])
        }

        for (const a in DNSAppType) {
            this.appTypes.push({ name: this._getDNSAppName(<any>a), value: DNSAppType[a] })
        }

        this._restoreZonesTableRowsPerPage()

        // Manage RxJS subscriptions on init.
        this._subscriptions = this._zonesTableFilter$
            .pipe(
                map((f) => {
                    return { ...f, value: f.value ?? null }
                }),
                debounceTime(300),
                distinctUntilChanged(),
                map((f) => {
                    f.filterConstraint.value = f.value
                    this.zone.run(() =>
                        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.zonesTable) })
                    )
                })
            )
            .subscribe()

        this.refreshFetchStatusTable()
    }

    /**
     * Component lifecycle hook which destroys the component.
     */
    ngOnDestroy() {
        this._subscriptions.unsubscribe()
        this._zonesTableFilter$.complete()
    }

    /**
     * Fetches data from backend and refreshes Zones Fetch Status table with the data.
     * If zones fetch is in progress, it subscribes to _polling$ observable to receive and
     * visualize Fetch progress.
     */
    refreshFetchStatusTable() {
        this.zonesFetchStatesLoading = true
        lastValueFrom(this.getZonesFetchWithStatus())
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
                        this._resetZoneInventoryStateMap()

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
            .finally(() => (this.zonesFetchStatesLoading = false))
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
     * Resets zoneInventoryStateMap with new values from zonesFetchStates.
     * @private
     */
    private _resetZoneInventoryStateMap() {
        if (this.zonesFetchStates.length > 0) {
            this.zoneInventoryStateMap = new Map()
            this.zonesFetchStates.forEach((s) => {
                this.zoneInventoryStateMap.set(s.daemonId, s)
            })
        }
    }

    /**
     * Sends PUT /dns-management/zones-fetch request and triggers refreshing data of the Zones Fetch Status table right after.
     * @private
     */
    private _sendPutZonesFetch() {
        this._subscriptions.add(this._putZonesFetchGuard.subscribe())
        this.fetchAppsCompletedCount = 0

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
                this._storeZoneFetchSent(true)
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
                    detail: 'Failed to start fetching zones: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Reference to getSeverity() function so it can be used in the html template.
     * @protected
     */
    protected readonly getSeverity = getSeverity

    /**
     * Returns more verbose error message for given error.
     * @param err error message received from backend
     * @return error message
     */
    getStateErrorMessage(err: string) {
        return `Error when communicating with a zone inventory on an agent: ${err}.`
    }

    /**
     * Reference to getTooltip() function so it can be used in the html template.
     * @protected
     */
    protected readonly getTooltip = getTooltip

    /**
     * Lazily loads paged zones data from backend.
     * @param event PrimeNG TableLazyLoadEvent with metadata about table pagination.
     * @param showLoadingState when set to false, zones table will not show loading state when data is lazily loaded from backend.
     *                         It is useful when zones fetch is in progress and table data is refreshed every polling interval - it
     *                         prevents table UI from flickering.
     *                         Defaults to true.
     */
    onLazyLoadZones(event: TableLazyLoadEvent, showLoadingState: boolean = true) {
        this.zonesLoading = showLoadingState
        this.cd.detectChanges() // in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        lastValueFrom(
            this.dnsService
                .getZones(
                    event?.first ?? 0,
                    event?.rows ?? 10,
                    (event?.filters?.appType as FilterMetadata)?.value ?? null,
                    (event?.filters?.zoneType as FilterMetadata)?.value ?? null,
                    (event?.filters?.zoneClass as FilterMetadata)?.value ?? null,
                    (event?.filters?.text as FilterMetadata)?.value || null,
                    (event?.filters?.appId as FilterMetadata)?.value || null,
                    (event?.filters?.zoneSerial as FilterMetadata)?.value || null,
                    this._getRPZFilterValue((event?.filters?.rpz as FilterMetadata)?.value)
                )
                .pipe(
                    map((resp) => {
                        resp.items?.forEach((zone) => {
                            let elz: ExtendedLocalZone[] = []
                            zone.localZones.forEach((localZone) => {
                                elz.push({ ...localZone, disableShowZone: this._shouldDisableShowZone(localZone) })
                            })
                            zone.localZones = elz
                        })
                        return resp
                    })
                )
        )
            .then((resp) => {
                this.zones = resp?.items ?? []
                this.zonesTotal = resp?.total ?? 0
                if (!this.zonesTotal) {
                    this.zonesExpandedRows = {}
                }
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
     * Returns true if the "show zone" button should be disabled for the given
     * instance of the zone on a server.
     *
     * @param localZone instance of the zone on a server
     * @returns true if the "show zone" button should be disabled for the given
     * instance of the zone on a server
     */
    private _shouldDisableShowZone(localZone: LocalZone): boolean {
        const allowedTypes: string[] = [DNSZoneType.Primary, DNSZoneType.Secondary, 'master', 'slave']
        return !allowedTypes.includes(localZone.zoneType)
    }

    /**
     * Retrieves information from browser storage whether PUT /dns-management/zones-fetch was sent or not.
     * @return boolean flag
     */
    wasZoneFetchSent(): boolean {
        const fromStorage = sessionStorage.getItem(this._fetchSentStorageKey) ?? 'false'
        return JSON.parse(fromStorage) === true
    }

    /**
     * Stores information in browser storage whether PUT /dns-management/zones-fetch was sent or not.
     * @param sent request was sent or not
     * @private
     */
    private _storeZoneFetchSent(sent: boolean) {
        sessionStorage.setItem(this._fetchSentStorageKey, JSON.stringify(sent))
    }

    /**
     * Reference to hasFilter() utility function so it can be used in the html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Asynchronously provides DNS zone by given zone ID.
     * @param id zone ID
     */
    zoneProvider: (id: number) => Promise<Zone> = (id) => lastValueFrom(this.dnsService.getZone(id))

    /**
     * Provides tab title for given zone.
     * @param zone zone for which the title is computed
     */
    tabTitleProvider: (entity: Zone) => string = (zone: Zone) => unrootZone(zone.name)

    /**
     * Resets zones table state and reloads the table without any filters applied.
     */
    clearTableState() {
        this.zonesTable?.clear()
        this.zone.run(() => this.router.navigate([]))
    }

    /**
     * Emits next value and filterConstraint for the zones table's filter,
     * which in the end will result in applying the filter on the table's data.
     * @param value value of the filter
     * @param filterConstraint filter field which will be filtered
     */
    filterZonesTable(value: any, filterConstraint: FilterMetadata): void {
        this._zonesTableFilter$.next({ value, filterConstraint })
    }

    /**
     * Clears a value for given zone table filter constraint and reloads the table with the new filtering.
     * @param filterConstraint
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.zone.run(() => this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.zonesTable) }))
    }

    /**
     * Stores only rows per page count for the zones table in user browser storage.
     */
    storeZonesTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.zonesTable?.getStorage()
        storage?.setItem(this._zonesTableStateStorageKey, JSON.stringify(state))
    }

    /**
     * Keeps track whether builtin zones are filtered out or not.
     */
    get builtinZonesDisplayed(): boolean {
        const selectedZoneTypes = (<FilterMetadata>this.zonesTable?.filters?.['zoneType'])?.value ?? []
        if (selectedZoneTypes.length === 0 || selectedZoneTypes.length === this.zoneTypes.length) {
            // Filtering with no zone types selected has the same effect as filtering with all zone types selected.
            // In this case builtin zones are also displayed.
            return true
        }

        return selectedZoneTypes.includes('builtin')
    }

    /**
     * Toggles filter in/filter out builtin zones by manipulating the zoneType filter.
     */
    toggleBuiltinZones() {
        // Get current zoneType filter.
        const zoneTypeFilterMetadata = <FilterMetadata>this.zonesTable?.filters?.['zoneType']

        if (!zoneTypeFilterMetadata) {
            // This shouldn't happen. But if there is no zoneType filter, simply return.
            return
        }

        const selectedZoneTypes: string[] = zoneTypeFilterMetadata.value ?? []

        if (this.builtinZonesDisplayed && selectedZoneTypes.length <= 1) {
            // Builtin zones are displayed because no zoneType filter is applied at all.
            // Change the filter empty value to all values selected, so that only builtin zones will be filtered out later.
            zoneTypeFilterMetadata.value = this.zoneTypes
        } else if (!this.builtinZonesDisplayed && selectedZoneTypes.length === this.zoneTypes.length - 1) {
            // zoneType filter is applied and builtin zones is the only type that is filtered out.
            // Adding builtin zones filter means filtering by all types, so the zoneType filter can be cleared.
            this.clearFilter(zoneTypeFilterMetadata)
            return
        }

        // Add or remove builtin zones to current zoneType filter.
        zoneTypeFilterMetadata.value = this.builtinZonesDisplayed
            ? zoneTypeFilterMetadata.value.filter((type) => type !== 'builtin')
            : [...zoneTypeFilterMetadata.value, 'builtin']

        // Apply filters to the zones table.
        this.zone.run(() => this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.zonesTable) }))
    }

    /**
     * Creates a unique key for the zone viewer dialog visibility map.
     *
     * @param daemonId daemon ID.
     * @param viewName view name.
     * @param zoneId zone ID.
     * @returns unique key.
     */
    private _getZoneViewerKey(daemonId: number, viewName: string, zoneId: number): string {
        return `${daemonId}:${viewName}:${zoneId}`
    }

    /**
     * Sets the visibility state of the zone viewer dialog.
     *
     * @param daemonId daemon ID.
     * @param viewName view name.
     * @param zoneId zone ID.
     * @param visible visibility state.
     */
    setZoneViewerDialogVisible(daemonId: number, viewName: string, zoneId: number, visible: boolean) {
        const key = this._getZoneViewerKey(daemonId, viewName, zoneId)
        this.zoneViewerDialogVisible.set(key, visible)
    }

    /**
     * Returns the visibility state of the zone viewer dialog.
     *
     * @param daemonId daemon ID.
     * @param viewName view name.
     * @param zoneId zone ID.
     * @returns visibility state.
     */
    getZoneViewerDialogVisible(daemonId: number, viewName: string, zoneId: number): boolean {
        const key = this._getZoneViewerKey(daemonId, viewName, zoneId)
        return this.zoneViewerDialogVisible.get(key) ?? false
    }

    /**
     * Returns unique zone types for a given zone
     * @param zone Zone to get types from
     */
    getUniqueZoneTypes(zone: Zone): string[] {
        if (!zone?.localZones?.length) {
            return []
        }
        return [...new Set(zone.localZones.map((lz) => lz.zoneType))]
    }

    /**
     * Gets serial information for a zone
     * @param zone Zone to analyze
     * @returns Object containing serial and mismatch flag
     */
    getZoneSerialInfo(zone: Zone): { serial: string; hasMismatch: boolean } {
        if (!zone?.localZones?.length) {
            return { serial: 'N/A', hasMismatch: false }
        }

        const serials = zone.localZones.map((lz) => lz.serial)
        const uniqueSerials = [...new Set(serials)]

        return {
            serial: uniqueSerials[0]?.toString() ?? 'N/A',
            hasMismatch: uniqueSerials.length > 1,
        }
    }

    /**
     * Checks if any of the instances of the zone are response policy zones.
     *
     * @param zone Zone to check.
     * @returns true if any of the instances of the zone are response policy zones,
     * false otherwise.
     */
    includesRPZ(zone: Zone): boolean {
        return zone?.localZones?.some((lz) => lz.rpz)
    }

    /**
     * Returns the RPZ filter value for the zones table.
     *
     * @param selection RPZ filter selection in the dropdown.
     * @returns RPZ filter value sent to the backend.
     */
    private _getRPZFilterValue(selection: string | null): boolean | null {
        switch (selection) {
            case 'exclude':
                return false
            case 'only':
                return true
            default:
                return null
        }
    }
}
