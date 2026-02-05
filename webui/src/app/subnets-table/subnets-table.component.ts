import { Component, computed, effect, Input, OnDestroy, OnInit, signal, ViewChild } from '@angular/core'
import { convertSortingFields, tableFiltersToQueryParams, tableHasFilter } from '../table'
import { DHCPService, Subnet, SubnetSortField } from '../backend'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { MessageService, TableState, PrimeTemplate, MenuItem } from 'primeng/api'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'
import { getErrorMessage, getGrafanaSubnetTooltip, getGrafanaUrl } from '../utils'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SubnetWithUniquePools,
    extractUniqueSubnetPools,
} from '../subnets'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { NgIf, NgFor, DecimalPipe } from '@angular/common'
import { FloatLabel } from 'primeng/floatlabel'
import { InputNumber } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { Select } from 'primeng/select'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputText } from 'primeng/inputtext'
import { Tooltip } from 'primeng/tooltip'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { PoolBarsComponent } from '../pool-bars/pool-bars.component'
import { Message } from 'primeng/message'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { SplitButton } from 'primeng/splitbutton'
import { DaemonFilterComponent } from '../daemon-filter/daemon-filter.component'

@Component({
    selector: 'app-subnets-table',
    templateUrl: './subnets-table.component.html',
    styleUrl: './subnets-table.component.sass',
    imports: [
        Button,
        RouterLink,
        ManagedAccessDirective,
        TableModule,
        NgIf,
        PrimeTemplate,
        FloatLabel,
        InputNumber,
        FormsModule,
        Select,
        IconField,
        InputIcon,
        InputText,
        Tooltip,
        SubnetBarComponent,
        NgFor,
        HumanCountComponent,
        PoolBarsComponent,
        Message,
        DecimalPipe,
        PluralizePipe,
        EntityLinkComponent,
        TableCaptionComponent,
        SplitButton,
        DaemonFilterComponent,
    ],
})
export class SubnetsTableComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG table instance.
     */
    @ViewChild('subnetsTable') table: Table

    /**
     * URL to grafana.
     */
    @Input() grafanaUrl: string

    /**
     * ID of the DHCPv4 dashboard in Grafana.
     */
    @Input() grafanaDhcp4DashboardId: string

    /**
     * ID of the DHCPv6 dashboard in Grafana.
     */
    @Input() grafanaDhcp6DashboardId: string

    /**
     * Indicates if the data is being fetched from the server.
     */
    @Input() dataLoading: boolean = false

    /**
     * Collection of subnets currently displayed in the table.
     */
    dataCollection: SubnetWithUniquePools[] = []

    /**
     * Total number of subnets currently displayed in the table.
     */
    totalRecords: number = 0

    /**
     * Returns true if the table filtering does not exclude IPv6 subnets.
     */
    ipV6SubnetsFilterIncluded = signal<boolean>(true)

    /**
     * Keeps value for colspan attribute for the table "empty message" placeholder.
     */
    emptyMessageColspan = computed<number>(() => (this.ipV6SubnetsFilterIncluded() ? 12 : 9))

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

    /**
     * Menu items of the splitButton which appears only for narrower viewports in the filtering toolbar.
     */
    toolbarButtons: MenuItem[] = []

    /**
     * This flag states whether user has privileges to create new subnets.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canCreateSubnet = signal<boolean>(false)

    /**
     * Effect signal reacting on user privileges changes and triggering update of the splitButton model
     * inside the filtering toolbar.
     */
    privilegesChangeEffect = effect(() => {
        if (this.canCreateSubnet()) {
            this._updateToolbarButtons()
        }
    })

    /**
     * Updates filtering toolbar splitButton menu items.
     * Based on user privileges some menu items may be disabled or not.
     * @private
     */
    private _updateToolbarButtons() {
        const buttons: MenuItem[] = [
            {
                label: 'New Subnet',
                icon: 'pi pi-plus',
                routerLink: '/dhcp/subnets/new',
                disabled: !this.canCreateSubnet(),
            },
        ]
        this.toolbarButtons = [...buttons]
    }

    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private router: Router
    ) {}

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for subnets filtering. If it is
     * not specified, the current values are used when available.
     */
    loadData(event: TableLazyLoadEvent) {
        // Indicate that subnets refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getSubnets(JSON.stringify(event))

        this.ipV6SubnetsFilterIncluded.set((<FilterMetadata>event.filters['dhcpVersion'])?.value !== '4')

        lastValueFrom(
            this.dhcpApi
                .getSubnets(
                    event.first,
                    event.rows,
                    (event.filters['daemonId'] as FilterMetadata)?.value ?? null,
                    (event.filters['subnetId'] as FilterMetadata)?.value ?? null,
                    (event.filters['dhcpVersion'] as FilterMetadata)?.value ?? null,
                    (event.filters['text'] as FilterMetadata)?.value || null,
                    ...convertSortingFields<SubnetSortField>(event)
                )
                // Custom parsing for statistics
                .pipe(
                    map((subnets) => {
                        if (subnets.items) {
                            parseSubnetsStatisticValues(subnets.items)
                        }
                        return subnets
                    })
                )
        )
            .then((data) => {
                this.dataCollection = data.items ? extractUniqueSubnetPools(data.items) : []
                this.totalRecords = data.total ?? 0
            })
            .catch((error) => {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot load subnets',
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

        this._updateToolbarButtons()
    }

    /**
     * Returns true if the subnet list presents at least one IPv6 subnet.
     */
    get isAnyIPv6SubnetVisible(): boolean {
        return !!this.dataCollection?.some((s) => s.subnet.includes(':'))
    }

    /**
     * Returns true if the subnet list presents at least one subnet with name.
     */
    get isAnySubnetWithNameVisible(): boolean {
        return !!this.dataCollection?.some((s) => s.localSubnets?.some((ls) => !!ls.userContext?.['subnet-name']))
    }

    /**
     * Checks if the local subnets in a given subnet have different local
     * subnet IDs.
     *
     * All local subnet IDs should be the same for a given subnet. Otherwise,
     * it indicates a misconfiguration issue.
     *
     * @param subnet Subnet with local subnets
     * @returns True if the referenced local subnets have different IDs.
     */
    hasAssignedMultipleKeaSubnetIds(subnet: Subnet): boolean {
        const localSubnets = subnet.localSubnets
        if (!localSubnets || localSubnets.length <= 1) {
            return false
        }

        const firstId = localSubnets[0].id
        return localSubnets.slice(1).some((ls) => ls.id !== firstId)
    }

    /**
     * Checks if the local subnets in a given subnet have different subnet
     * names in their user context.
     *
     * @param subnet Subnet with local subnets
     * @returns True if the referenced local subnets have different subnet
     *          names.
     */
    hasAssignedMultipleSubnetNames(subnet: Subnet): boolean {
        const localSubnets = subnet.localSubnets
        if (!localSubnets || localSubnets.length <= 1) {
            return false
        }

        const firstSubnetName = localSubnets[0].userContext?.['subnet-name']
        return localSubnets.slice(1).some((ls) => ls.userContext?.['subnet-name'] !== firstSubnetName)
    }

    /**
     * Get total number of addresses in a subnet.
     */
    getTotalAddresses(subnet: Subnet) {
        return getTotalAddresses(subnet)
    }

    /**
     * Get assigned number of addresses in a subnet.
     */
    getAssignedAddresses(subnet: Subnet) {
        return getAssignedAddresses(subnet)
    }

    /**
     * Get total number of delegated prefixes in a subnet.
     */
    getTotalDelegatedPrefixes(subnet: Subnet) {
        return subnet.stats?.['total-pds']
    }

    /**
     * Get assigned number of delegated prefixes in a subnet.
     */
    getAssignedDelegatedPrefixes(subnet: Subnet) {
        return subnet.stats?.['assigned-pds']
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        let dashboardId = ''
        if (name === 'dhcp4') {
            dashboardId = this.grafanaDhcp4DashboardId
        } else if (name === 'dhcp6') {
            dashboardId = this.grafanaDhcp6DashboardId
        }

        return getGrafanaUrl(this.grafanaUrl, dashboardId, subnet, instance)
    }

    /**
     * Builds a tooltip explaining what the link is for.
     * @param subnet an identifier of the subnet
     * @param machine an identifier of the machine the subnet is configured on
     */
    getGrafanaTooltip(subnet: number, machine: string) {
        return getGrafanaSubnetTooltip(subnet, machine)
    }

    /**
     * Reference to the function so it can be used in html template.
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
    private readonly _tableStateStorageKey = 'subnets-table-state'

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
    protected readonly SubnetSortField = SubnetSortField

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
