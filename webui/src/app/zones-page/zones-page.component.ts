import { AfterViewInit, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ConfirmationService, MenuItem, MessageService, TableState } from 'primeng/api'
import {
    DNSAppType,
    DNSClass,
    DNSService,
    DNSZoneType,
    Zone,
    ZoneInventoryState,
    ZoneInventoryStates,
    ZonesFetchStatus,
} from '../backend'
import { TabViewCloseEvent } from 'primeng/tabview'
import {
    catchError,
    concatMap,
    delay,
    distinctUntilChanged,
    filter,
    finalize,
    map,
    share,
    switchMap,
    takeWhile,
    tap,
} from 'rxjs/operators'
import { debounceTime, EMPTY, interval, lastValueFrom, of, Subject, Subscription, timer } from 'rxjs'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { getErrorMessage } from '../utils'
import { HttpResponse, HttpStatusCode } from '@angular/common/http'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { hasFilter, parseBoolean } from '../table'
import { ActivatedRoute, ParamMap } from '@angular/router'
import StatusEnum = ZoneInventoryState.StatusEnum

/**
 * Returns tooltip message for given ZoneInventoryState status.
 * @param status ZoneInventoryState status
 */
export function getTooltip(status: StatusEnum) {
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
 * Returns PrimeNG severity for given ZoneInventoryState status.
 * @param status ZoneInventoryState status
 */
export function getSeverity(status: StatusEnum) {
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

@Component({
    selector: 'app-zones-page',
    templateUrl: './zones-page.component.html',
    styleUrl: './zones-page.component.sass',
})
export class ZonesPageComponent implements OnInit, OnDestroy, AfterViewInit {
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
     * Key to be used in browser storage for keeping zones table state.
     * @private
     */
    private readonly _zonesTableStateStorageKey = 'zones-table-state'

    /**
     * Key to be used for dynamic binding to stateKey input property of zones PrimeNG table.
     * Changing this value will have effect on whether zones table is stateful or not.
     */
    zonesStateKey: string = this._zonesTableStateStorageKey

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
    private _isPolling = false

    /**
     * RxJS Subject used for filtering zones table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _zonesTableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * Class constructor.
     * @param cd Angular change detection required to manually trigger detectChanges in this component
     * @param dnsService service providing DNS REST APIs
     * @param messageService PrimeNG message service used to display feedback messages in UI
     * @param confirmationService PrimeNG confirmation service used to display confirmation dialog
     * @param activatedRoute Angular ActivatedRoute to retrieve information about current route queryParams
     */
    constructor(
        private cd: ChangeDetectorRef,
        private dnsService: DNSService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService,
        private activatedRoute: ActivatedRoute
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
     * DNS app types values used for the UI filter dropdown options.
     */
    appTypes: { name: string; value: string }[] = []

    /**
     * Boolean flag stating whether zones table should lazily load zones from backend on component init.
     */
    loadZonesOnInit = true

    /**
     * Returns label for the DNS App type.
     * @param appType DNS App type
     */
    private _getDNSAppName(appType: DNSAppType) {
        switch (appType) {
            case DNSAppType.Bind9:
                return 'BIND9'
            // case DNSAppType.Pdns:
            //     return 'PowerDNS'
            default:
                return (<string>appType).toUpperCase()
        }
    }

    /**
     * Component lifecycle hook which executes after the component view was initialized.
     * It is likely that filtering shall be done in PrimeNG table. Managing the filtering
     * at this step is safe because all child components (also the PrimeNG table itself)
     * should be initialized.
     */
    ngAfterViewInit() {
        this._initDone = true
        if (!this.loadZonesOnInit) {
            // Valid zones filter was provided via URL queryParams.
            this._restoreZonesTableRowsPerPage()
            this._filterZonesByQueryParams()
        }
    }

    /**
     * Keeps current valid zone filters parsed from URL queryParams.
     * The type of this object is inline with PrimeNG table filters property.
     */
    queryParamFilters: { [p: string]: FilterMetadata } = {}

    /**
     * Object containing supported zone filters which values are provided via URL deep-link.
     * The properties of this object correspond to queryParam keys.
     * Values of this object describe:
     * - filter type (numeric, enum, string or boolean)
     * - filter matchMode (contains, equals) which corresponds to PrimeNG table filter metadata
     * - accepted enum values for enum type of filters
     * - array type; when set to true it means that the filter may use more than one value.
     * @private
     */
    private _supportedQueryParamFilters: {
        [k: string]: {
            type: 'numeric' | 'enum' | 'string' | 'boolean'
            matchMode: 'contains' | 'equals'
            enumValues?: string[]
            arrayType?: boolean
        }
    } = {
        appId: { type: 'numeric', matchMode: 'contains' },
        appType: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSAppType) },
        zoneType: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSZoneType), arrayType: true },
        zoneClass: { type: 'enum', matchMode: 'equals', enumValues: Object.values(DNSClass) },
        text: { type: 'string', matchMode: 'contains' },
        zoneSerial: { type: 'string', matchMode: 'contains' },
    }

    /**
     * Parses zone filters from given URL queryParamMap, validates them and applies to queryParamFilters property.
     * Filter validation relies on correctly initialized _supportedQueryParamFilters property.
     * Returns number of valid filters found.
     * @param queryParamMap URL queryParamMap that will be used for zone filters parsing
     */
    parseQueryParams(queryParamMap: ParamMap): number {
        let validFilters = 0
        this.queryParamFilters = {}
        for (let paramKey of queryParamMap.keys) {
            if (!(paramKey in this._supportedQueryParamFilters)) {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Wrong URL parameter value',
                    detail: `URL parameter ${paramKey} not supported!`,
                    life: 10000,
                })
                continue
            }

            const paramValues = this._supportedQueryParamFilters[paramKey].arrayType
                ? queryParamMap.getAll(paramKey)
                : [queryParamMap.get(paramKey)]
            for (let paramValue of paramValues) {
                if (paramValue) {
                    let parsedValue = null
                    switch (this._supportedQueryParamFilters[paramKey].type) {
                        case 'numeric':
                            const numV = parseInt(paramValue, 10)
                            if (Number.isNaN(numV)) {
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Wrong URL parameter value',
                                    detail: `URL parameter ${paramKey} requires numeric value!`,
                                    life: 10000,
                                })
                                break
                            }

                            parsedValue = numV
                            validFilters += 1
                            break
                        case 'boolean':
                            const booleanV = parseBoolean(paramValue)
                            if (booleanV === null) {
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Wrong URL parameter value',
                                    detail: `URL parameter ${paramKey} requires either true or false value!`,
                                    life: 10000,
                                })
                                break
                            }

                            parsedValue = booleanV
                            validFilters += 1
                            break
                        case 'enum':
                            if ((this._supportedQueryParamFilters[paramKey].enumValues ?? []).length === 0) {
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Wrong URL parameter value',
                                    detail: `URL parameter ${paramKey} of type ${this._supportedQueryParamFilters[paramKey].type} not supported!`,
                                    life: 10000,
                                })
                                break
                            }

                            if (this._supportedQueryParamFilters[paramKey].enumValues?.includes(paramValue)) {
                                parsedValue = paramValue
                                validFilters += 1
                                break
                            }

                            this.messageService.add({
                                severity: 'error',
                                summary: 'Wrong URL parameter value',
                                detail: `URL parameter ${paramKey} requires one of the values: ${this._supportedQueryParamFilters[paramKey].enumValues.join(', ')}!`,
                                life: 10000,
                            })
                            break
                        case 'string':
                            parsedValue = paramValue
                            validFilters += 1
                            break
                        default:
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Wrong URL parameter value',
                                detail: `URL parameter ${paramKey} of type ${this._supportedQueryParamFilters[paramKey].type} not supported!`,
                                life: 10000,
                            })
                            break
                    }

                    if (parsedValue !== null) {
                        const filterConstraint = {}
                        if (this._supportedQueryParamFilters[paramKey].arrayType) {
                            parsedValue = this.queryParamFilters[paramKey]?.value
                                ? [...this.queryParamFilters[paramKey]?.value, parsedValue]
                                : [parsedValue]
                        }

                        filterConstraint[paramKey] = {
                            value: parsedValue,
                            matchMode: this._supportedQueryParamFilters[paramKey].matchMode,
                        }
                        this.queryParamFilters = { ...this.queryParamFilters, ...filterConstraint }
                    }
                }
            }
        }

        return validFilters
    }

    /**
     * Boolean flag stating whether this component init is done or not.
     * @private
     */
    private _initDone = false

    /**
     * Makes the zones table stateful, which means that pagination, filtering, sorting etc. will be stored
     * in user browser storage.
     * @private
     */
    private _enableStatefulZonesTable() {
        this.zonesStateKey = this._zonesTableStateStorageKey
        this.cd.detectChanges()
    }

    /**
     * Makes the zones table NOT stateful, which means that pagination, filtering, sorting etc. will NOT be stored
     * in user browser storage.
     * @private
     */
    private _disableStatefulZonesTable() {
        this.zonesStateKey = null
    }

    /**
     * Restores only rows per page count for the zones table from the state stored in user browser storage.
     * @private
     */
    private _restoreZonesTableRowsPerPage() {
        const storage = this.zonesTable?.getStorage()
        const stateString = storage?.getItem(this._zonesTableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.zonesTable.rows = state.rows ?? 10
        }
    }

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        // Initialize arrays that contain values for UI filter dropdowns.
        for (let t in DNSZoneType) {
            this.zoneTypes.push(DNSZoneType[t])
        }

        for (let c in DNSClass) {
            this.zoneClasses.push(DNSClass[c])
        }

        for (let a in DNSAppType) {
            this.appTypes.push({ name: this._getDNSAppName(<any>a), value: DNSAppType[a] })
        }

        // Manage RxJS subscriptions on init.
        this._subscriptions = this.activatedRoute.queryParamMap
            .pipe(filter(() => this._initDone))
            .subscribe((value) => {
                const queryParamFiltersCount = this.parseQueryParams(value)
                if (queryParamFiltersCount > 0) {
                    // Disable stateful zones table when filtering via URL queryParams is in place.
                    this._disableStatefulZonesTable()
                    this._filterZonesByQueryParams()
                    if (this.activeTabIdx > 0) {
                        // Go back to first tab with zones list.
                        this.activateFirstTab()
                    }
                    return
                }

                this._enableStatefulZonesTable()
                // URL queryParams changed, but no valid filter was found there so force restore table state.
                this.zonesTable?.restoreState()
                this.zonesTable?._filter()
            })
        this._subscriptions.add(
            this._zonesTableFilter$
                .pipe(
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        this.zonesTable?._filter()
                    })
                )
                .subscribe()
        )

        const queryParamFiltersCount = this.parseQueryParams(this.activatedRoute.snapshot.queryParamMap)
        if (queryParamFiltersCount > 0) {
            // Valid filters found, so do not load lazily zones on init, because zones with appropriate filters
            // will be loaded later.
            this.loadZonesOnInit = false
            // Disable stateful zones table when filtering via URL queryParams is in place.
            this._disableStatefulZonesTable()
            this.zonesLoading = true
        }

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
     * Callback called when detailed zone tab is closed.
     * @param event closing event
     */
    onTabClose(event: TabViewCloseEvent) {
        this.openTabs.splice(event.index - 1, 1)
        this.cd.detectChanges()
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
     */
    onLazyLoadZones(event: TableLazyLoadEvent) {
        this.zonesLoading = true
        this.cd.detectChanges() // in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        lastValueFrom(
            this.dnsService.getZones(
                event?.first ?? 0,
                event?.rows ?? 10,
                (event?.filters?.appType as FilterMetadata)?.value ?? null,
                (event?.filters?.zoneType as FilterMetadata)?.value ?? null,
                (event?.filters?.zoneClass as FilterMetadata)?.value ?? null,
                (event?.filters?.text as FilterMetadata)?.value || null,
                (event?.filters?.appId as FilterMetadata)?.value || null,
                (event?.filters?.zoneSerial as FilterMetadata)?.value || null
            )
        )
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
     * Retrieves information from browser storage whether PUT /dns-management/zones-fetch was sent or not.
     */
    wasZoneFetchSent(): boolean {
        const fromStorage = sessionStorage.getItem(this._fetchSentStorageKey) ?? 'false'
        return JSON.parse(fromStorage) === true
    }

    /**
     * Stores information in browser storage whether PUT /dns-management/zones-fetch was sent or not.
     * @param sent request was sent or not
     */
    storeZoneFetchSent(sent: boolean) {
        sessionStorage.setItem(this._fetchSentStorageKey, JSON.stringify(sent))
    }

    /**
     * Reference to hasFilter() utility function so it can be used in the html template.
     * @protected
     */
    protected readonly hasFilter = hasFilter

    /**
     * Resets zones table state and updates the state stored in browser storage.
     */
    clearTableState() {
        this.zonesTable?.clear()
        this.zonesTable?.saveState()
    }

    /**
     * Emits next value and filterConstraint for the zones table's filter,
     * which in the end will result in applying the filter on the table's data.
     * @param value value of the filter
     * @param filterConstraint filter field which will be filtered
     */
    filterTable(value: any, filterConstraint: FilterMetadata): void {
        this._zonesTableFilter$.next({ value, filterConstraint })
    }

    /**
     * Filters zones table by filters provided via URL queryParams.
     * @private
     */
    private _filterZonesByQueryParams(): void {
        this.zonesTable?.clearFilterValues()
        const metadata = this.zonesTable?.createLazyLoadMetadata()
        this.zonesTable.filters = { ...metadata.filters, ...this.queryParamFilters }
        this.zonesTable?._filter()
    }

    /**
     * Activates the first tab in the view with the zones table.
     */
    activateFirstTab() {
        this.activeTabIdx = 0
    }
}
