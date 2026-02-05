import { Component, effect, OnDestroy, OnInit, signal, ViewChild } from '@angular/core'
import { tableHasFilter, tableFiltersToQueryParams, convertSortingFields } from '../table'
import { DHCPService, Host, HostSortField, LocalHost } from '../backend'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { ConfirmationService, MenuItem, MessageService, PrimeTemplate, TableState } from 'primeng/api'
import { getErrorMessage, uncamelCase } from '../utils'
import { hasDifferentLocalHostData } from '../hosts'
import { debounceTime, last, lastValueFrom, Subject, Subscription } from 'rxjs'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { ManagedAccessDirective } from '../managed-access.directive'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { NgFor, NgIf } from '@angular/common'
import { Button } from 'primeng/button'
import { FormsModule } from '@angular/forms'
import { FloatLabel } from 'primeng/floatlabel'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputNumber } from 'primeng/inputnumber'
import { InputText } from 'primeng/inputtext'
import { Tag } from 'primeng/tag'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HostDataSourceLabelComponent } from '../host-data-source-label/host-data-source-label.component'
import { IdentifierComponent } from '../identifier/identifier.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { Tooltip } from 'primeng/tooltip'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { SplitButton } from 'primeng/splitbutton'
import { DaemonFilterComponent } from '../daemon-filter/daemon-filter.component'

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
    templateUrl: './hosts-table.component.html',
    styleUrls: ['./hosts-table.component.sass'],
    imports: [
        ManagedAccessDirective,
        ConfirmDialog,
        NgFor,
        Button,
        RouterLink,
        TableModule,
        NgIf,
        Tag,
        PrimeTemplate,
        FloatLabel,
        InputNumber,
        FormsModule,
        TriStateCheckboxComponent,
        IconField,
        InputIcon,
        InputText,
        IdentifierComponent,
        EntityLinkComponent,
        HostDataSourceLabelComponent,
        Tooltip,
        PluralizePipe,
        TableCaptionComponent,
        SplitButton,
        DaemonFilterComponent,
    ],
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
                (event.filters['machineId'] as FilterMetadata)?.value ?? null,
                (event.filters['daemonId'] as FilterMetadata)?.value ?? null,
                (event.filters['subnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['keaSubnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['text'] as FilterMetadata)?.value || null,
                (event.filters['isGlobal'] as FilterMetadata)?.value ?? null,
                (event.filters['conflict'] as FilterMetadata)?.value ?? null,
                ...convertSortingFields<HostSortField>(event)
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
     * Holds local hosts of all currently displayed host reservations grouped by daemon ID.
     * It is indexed by host ID.
     */
    localHostsGroupedByDaemon: Record<number, LocalHost[][]>

    /**
     * This flag states whether user has privileges to start the migration.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canStartMigration = signal<boolean>(false)

    /**
     * This flag states whether user has privileges to create new host reservations.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canCreateHosts = signal<boolean>(false)

    /**
     * Effect signal reacting on user privileges changes and triggering update of the splitButton model
     * inside the filtering toolbar.
     */
    privilegesChangeEffect = effect(() => {
        if (this.canStartMigration() || this.canCreateHosts()) {
            this._updateToolbarButtons()
        }
    })

    /**
     * Returns all currently displayed host reservations.
     */
    get hosts(): Host[] {
        return this.dataCollection
    }

    /**
     * Sets hosts reservations to be displayed.
     * Groups the local hosts by daemon ID and stores the result in
     * @this.localHostsGroupedByDaemon.
     */
    set hosts(hosts: Host[]) {
        this.dataCollection = hosts

        // For each host group the local hosts by daemon ID.
        this.localHostsGroupedByDaemon = Object.fromEntries(
            (hosts || []).map((host) => {
                if (!host.localHosts) {
                    return [host.id, []]
                }

                return [
                    host.id,
                    Object.values(
                        // Group the local hosts by daemon ID.
                        host.localHosts.reduce<Record<number, LocalHost[]>>((accDaemon, localHost) => {
                            if (!accDaemon[localHost.daemonId]) {
                                accDaemon[localHost.daemonId] = []
                            }

                            accDaemon[localHost.daemonId].push(localHost)

                            return accDaemon
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
     * Menu items of the splitButton which appears only for narrower viewports in the filtering toolbar.
     */
    toolbarButtons: MenuItem[] = []

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
                    map((f) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                    debounceTime(300),
                    distinctUntilChanged()
                )
                .subscribe((f) => {
                    // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                    // so it's value must be set according to UI columnFilter value.
                    f.filterConstraint.value = f.value
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
                })
        )

        this._updateToolbarButtons()
    }

    /**
     * Displays a modal dialog with the details of the host migration.
     * The dialog displays the host filter and the total number of migrated
     * hosts. There is also warning that the related daemons will be locked
     * during the migration. User can confirm or abort the migration.
     */
    migrateToDatabaseAsk(): void {
        if (!this.canStartMigration()) {
            return
        }

        // Display a confirmation dialog.
        this.confirmationService.confirm({
            key: 'migrationToDatabaseDialog',
            header: 'Migrate host reservations to database',
            icon: 'pi pi-exclamation-triangle',
            rejectButtonProps: { icon: 'pi pi-times' },
            acceptButtonProps: {
                icon: 'pi pi-check',
            },
            accept: () => {
                // User confirmed the migration.
                this.dhcpApi
                    .startHostsMigration(
                        (this.table?.filters['machineId'] as FilterMetadata)?.value ?? null,
                        (this.table?.filters['daemonId'] as FilterMetadata)?.value ?? null,
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
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.table?.clearFilterValues()
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

    /**
     * Reference to an enum so it could be used in the HTML template.
     * @protected
     */
    protected readonly HostSortField = HostSortField

    /**
     * Updates filtering toolbar splitButton menu items.
     * Based on user privileges some menu items may be disabled or not.
     * @private
     */
    private _updateToolbarButtons() {
        const buttons: MenuItem[] = [
            {
                label: 'Migrate to Database',
                command: () => this.migrateToDatabaseAsk(),
                icon: 'pi pi-database',
                disabled: !this.canStartMigration(),
            },
            {
                label: 'New Host',
                routerLink: '/dhcp/hosts/new',
                icon: 'pi pi-plus',
                disabled: !this.canCreateHosts(),
            },
        ]
        this.toolbarButtons = [...buttons]
    }

    /**
     * Callback called when autocomplete form emits any error message.
     * @param message error message
     */
    onAutocompleteError(message: string) {
        this.messageService.add({
            severity: 'error',
            summary: 'Cannot get daemons directory',
            detail: message,
            life: 10000,
        })
    }
}
