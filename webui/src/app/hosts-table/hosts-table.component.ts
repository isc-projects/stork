import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { tableHasFilter, tableFiltersToQueryParams } from '../table'
import { DHCPService, Host, LocalHost } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { Router } from '@angular/router'
import { ConfirmationService, MessageService, TableState } from 'primeng/api'
import { getErrorMessage, uncamelCase } from '../utils'
import { hasDifferentLocalHostData } from '../hosts'
import { debounceTime, last, lastValueFrom, Subject, Subscription } from 'rxjs'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { distinctUntilChanged, map } from 'rxjs/operators'

/**
 * This component implements a table of hosts reservations.
 * The list of hosts is paged and can be filtered by provided
 * URL queryParams or by using form inputs responsible for
 * filtering. The list contains hosts reservations for all subnets
 * and also contain global reservations, i.e. those that are not
 * associated with any particular subnet.
 */
@Component({
    selector: 'app-hosts-table',
    standalone: false,
    templateUrl: './hosts-table.component.html',
    styleUrls: ['./hosts-table.component.sass'],
})
export class HostsTableComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG table instance.
     */
    @ViewChild('hostsTable') table: Table

    /**
     * Flag stating whether table data is loading or not.
     */
    dataLoading: boolean

    /**
     * Total number of records displayed currently in the table.
     */
    totalRecords: number = 0

    /**
     * Data collection displayed currently in the table.
     */
    dataCollection: Host[] = []

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

    constructor(
        private router: Router,
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService
    ) {}

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for hosts filtering. If it is
     * not specified, the current values are used when available.
     */
    loadData(event: TableLazyLoadEvent) {
        // Indicate that hosts refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getHosts(JSON.stringify(event))
        lastValueFrom(
            this.dhcpApi.getHosts(
                event.first,
                event.rows,
                (event.filters['appId'] as FilterMetadata)?.value ?? null,
                (event.filters['subnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['keaSubnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['text'] as FilterMetadata)?.value || null,
                (event.filters['isGlobal'] as FilterMetadata)?.value ?? null,
                (event.filters['conflict'] as FilterMetadata)?.value ?? null
            )
        )
            .then((data) => {
                this.hosts = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get host reservations list',
                    detail: 'Error getting host reservations list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    /**
     * Holds local hosts of all currently displayed host reservations grouped by app ID.
     * It is indexed by host ID.
     */
    localHostsGroupedByApp: Record<number, LocalHost[][]>

    /**
     * This flag states whether user has privileges to start the migration.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canStartMigration: boolean = false

    /**
     * Returns all currently displayed host reservations.
     */
    get hosts(): Host[] {
        return this.dataCollection
    }

    /**
     * Sets hosts reservations to be displayed.
     * Groups the local hosts by app ID and stores the result in
     * @this.localHostsGroupedByApp.
     */
    set hosts(hosts: Host[]) {
        this.dataCollection = hosts

        // For each host group the local hosts by app ID.
        this.localHostsGroupedByApp = Object.fromEntries(
            (hosts || []).map((host) => {
                if (!host.localHosts) {
                    return [host.id, []]
                }

                return [
                    host.id,
                    Object.values(
                        // Group the local hosts by app ID.
                        host.localHosts.reduce<Record<number, LocalHost[]>>((accApp, localHost) => {
                            if (!accApp[localHost.appId]) {
                                accApp[localHost.appId] = []
                            }

                            accApp[localHost.appId].push(localHost)

                            return accApp
                        }, {})
                    ),
                ]
            })
        )
    }

    /**
     * Returns the state of the local hosts from the same application/daemon.
     * The state is null if the host reservations are defined only in the
     * configuration file or host database. If they are defined in both places
     * the state is one of the following:
     * - duplicate - reservations have the same boot fields, client classes, and
     *               DHCP options
     * - conflict - reservations are configured differently.
     *
     * @param localHosts local hosts to be checked.
     */
    getLocalHostsState(localHosts: LocalHost[]): 'conflict' | 'duplicate' | null {
        if (localHosts.length <= 1) {
            return null
        }
        if (hasDifferentLocalHostData(localHosts)) {
            return 'conflict'
        } else {
            return 'duplicate'
        }
    }

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        this._restoreTableRowsPerPage()

        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    map((f) => {
                        return { ...f, value: f.value === '' ? null : f.value }
                    }),
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
                    })
                )
                .subscribe()
        )
    }

    /**
     * Displays a modal dialog with the details of the host migration.
     * The dialog displays the host filter and the total number of migrated
     * hosts. There is also warning that the related daemons will be locked
     * during the migration. User can confirm or abort the migration.
     */
    migrateToDatabaseAsk(): void {
        if (!this.canStartMigration) {
            return
        }

        // Display a confirmation dialog.
        this.confirmationService.confirm({
            key: 'migrationToDatabaseDialog',
            header: 'Migrate host reservations to database',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                // User confirmed the migration.
                this.dhcpApi
                    .startHostsMigration(
                        (this.table?.filters['appId'] as FilterMetadata)?.value ?? null,
                        (this.table?.filters['subnetId'] as FilterMetadata)?.value ?? null,
                        (this.table?.filters['keaSubnetId'] as FilterMetadata)?.value ?? null,
                        (this.table?.filters['text'] as FilterMetadata)?.value || null,
                        (this.table?.filters['isGlobal'] as FilterMetadata)?.value ?? null
                    )
                    .pipe(last())
                    .subscribe({
                        next: (result) => {
                            this.router.navigate(['/config-migrations/' + result.id])
                        },
                        error: (error) => {
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Cannot migrate host reservations',
                                detail: getErrorMessage(error),
                            })
                        },
                    })
            },
        })
    }

    /**
     * Returns entries of the table filter that will be used to migrate the
     * hosts. The keys are uncamelized and capitalized. The conflict key is
     * always false.
     */
    get migrationFilterEntries() {
        const filters = { ...this.table?.filters, ...{ conflict: { value: false } } }
        return Object.entries(filters)
            .filter(([, filterMetadata]) => (<FilterMetadata>filterMetadata).value != null)
            .map(([key, filterMetadata]) => [uncamelCase(key), filterMetadata.value.toString()])
            .sort(([key1], [key2]) => key1.localeCompare(key2))
    }

    /**
     * Returns true when there is filtering by hosts that are in conflict enabled; false otherwise.
     */
    isFilteredByConflict(): boolean {
        return (<FilterMetadata>this.table?.filters['conflict'])?.value === true
    }

    /**
     * Clears the PrimeNG table state (filtering, pagination are reset).
     */
    clearTableState() {
        this.table?.clear()
        this.router.navigate([])
    }

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     *
     * @param value
     * @param filterConstraint
     * @param debounceMode
     */
    filterTable(value: any, filterConstraint: FilterMetadata, debounceMode = true): void {
        if (debounceMode) {
            this._tableFilter$.next({ value, filterConstraint })
            return
        }

        filterConstraint.value = value
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
    }

    /**
     * Reference to the function so it can be used in html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'hosts-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.table?.getStorage()
        storage?.setItem(this._tableStateStorageKey, JSON.stringify(state))
    }

    /**
     * Restores only rows per page count for the table from the state stored in user browser storage.
     * @private
     */
    private _restoreTableRowsPerPage() {
        const stateString = localStorage.getItem(this._tableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.rows = state.rows ?? 10
        }
    }
}
