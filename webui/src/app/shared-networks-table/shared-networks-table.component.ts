import { Component, computed, Input, OnDestroy, OnInit, signal, ViewChild } from '@angular/core'
import { convertSortingFields, tableFiltersToQueryParams, tableHasFilter } from '../table'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SharedNetworkWithUniquePools,
    extractUniqueSharedNetworkPools,
} from '../subnets'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { DHCPService, NetworkSortField, SharedNetwork } from '../backend'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { MessageService, TableState, PrimeTemplate } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { Panel } from 'primeng/panel'
import { NgIf, NgFor, DecimalPipe } from '@angular/common'
import { Tag } from 'primeng/tag'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { FloatLabel } from 'primeng/floatlabel'
import { InputNumber } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { Select } from 'primeng/select'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputText } from 'primeng/inputtext'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-table',
    templateUrl: './shared-networks-table.component.html',
    styleUrl: './shared-networks-table.component.sass',
    imports: [
        Button,
        RouterLink,
        ManagedAccessDirective,
        TableModule,
        Panel,
        NgIf,
        Tag,
        HelpTipComponent,
        PrimeTemplate,
        FloatLabel,
        InputNumber,
        FormsModule,
        Select,
        IconField,
        InputIcon,
        InputText,
        EntityLinkComponent,
        HumanCountComponent,
        NgFor,
        SubnetBarComponent,
        DecimalPipe,
        PluralizePipe,
    ],
})
export class SharedNetworksTableComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG table instance.
     */
    @ViewChild('networksTable') table: Table

    /**
     * Indicates if the data is being fetched from the server.
     */
    @Input() dataLoading: boolean = false

    /**
     * Data collection of shared networks that are currently displayed in the table.
     */
    dataCollection: SharedNetworkWithUniquePools[] = []

    /**
     * Total number of shared networks that are currently displayed in the table.
     */
    totalRecords: number = 0

    /**
     * Returns true if the table filtering does not exclude IPv6 subnets.
     */
    ipV6SubnetsFilterIncluded = signal<boolean>(true)

    /**
     * Keeps value for colspan attribute for the table "empty message" placeholder.
     */
    emptyMessageColspan = computed<number>(() => (this.ipV6SubnetsFilterIncluded() ? 9 : 6))

    /**
     * RxJS subscriptions that may be all unsubscribed when the component gets destroyed.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

    constructor(
        private dhcpApi: DHCPService,
        private router: Router,
        private messageService: MessageService
    ) {}

    /**
     * Loads shared networks from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for shared networks filtering. If it is
     * not specified, the current values are used when available.
     */
    loadData(event: TableLazyLoadEvent): void {
        // Indicate that shared networks refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getSharedNetworks(JSON.stringify(event))

        this.ipV6SubnetsFilterIncluded.set((<FilterMetadata>event.filters['dhcpVersion'])?.value !== '4')

        lastValueFrom(
            this.dhcpApi
                .getSharedNetworks(
                    event.first,
                    event.rows,
                    (event.filters['daemonId'] as FilterMetadata)?.value ?? null,
                    (event.filters['dhcpVersion'] as FilterMetadata)?.value ?? null,
                    (event.filters['text'] as FilterMetadata)?.value || null,
                    ...convertSortingFields<NetworkSortField>(event)
                )
                .pipe(
                    map((sharedNetworks) => {
                        parseSubnetsStatisticValues(sharedNetworks.items)
                        sharedNetworks.items = extractUniqueSharedNetworkPools(sharedNetworks.items)
                        return sharedNetworks
                    })
                )
        )
            .then((data) => {
                this.dataCollection = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((error) => {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot load shared networks',
                    detail: getErrorMessage(error),
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
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
    }

    /**
     * Get the total number of addresses in the network.
     */
    getTotalAddresses(network: SharedNetwork) {
        return getTotalAddresses(network)
    }

    /**
     * Get the number of assigned addresses in the network.
     */
    getAssignedAddresses(network: SharedNetwork) {
        return getAssignedAddresses(network)
    }

    /**
     * Get the total number of delegated prefixes in the network.
     */
    getTotalDelegatedPrefixes(network: SharedNetwork) {
        return network.universe === 6 ? network.stats?.['total-pds'] : undefined
    }

    /**
     * Get the number of delegated prefixes in the network.
     */
    getAssignedDelegatedPrefixes(network: SharedNetwork) {
        return network.universe === 6 ? network.stats?.['assigned-pds'] : undefined
    }

    /**
     * Returns a list of daemons maintaining a given shared network.
     * The list doesn't contain duplicates.
     *
     * @param net Shared network
     * @returns List of the daemons (only ID and daemon label)
     */
    getDaemons(net: SharedNetwork): { id: number; label: string }[] {
        const daemons = []
        const daemonIds: Record<number, boolean> = {}

        net.localSharedNetworks?.forEach((lsn) => {
            const id = lsn.daemonId
            if (!id || daemonIds[id]) {
                return
            }
            const label = lsn.daemonLabel
            daemons.push({ id, label })
            daemonIds[id] = true
        })

        return daemons
    }

    /**
     * Reference to tableHasFilter function so that it can be used in the html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

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
     * Filters table data based on single UI filtering form input.
     * @param value value of the filter to be applied
     * @param filterConstraint PrimeNG table filter metadata to be set
     * @param debounceMode if set to true, the filtering is applied by RxJS subject _tableFilter$, which has debounceTime operator applied.
     *                      If set to false, the filtering is done immediately. Defaults to true.
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
    private readonly _tableStateStorageKey = 'networks-table-state'

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
    protected readonly NetworkSortField = NetworkSortField
}
