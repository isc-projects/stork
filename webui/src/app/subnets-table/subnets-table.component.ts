import { Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { DHCPService, Subnet } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { Router } from '@angular/router'
import { MessageService } from 'primeng/api'
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

@Component({
    selector: 'app-subnets-table',
    templateUrl: './subnets-table.component.html',
    styleUrl: './subnets-table.component.sass',
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
    dataCollection: SubnetWithUniquePools[] = []
    totalRecords: number = 0
    private _subscriptions: Subscription = new Subscription()

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

        lastValueFrom(
            this.dhcpApi
                .getSubnets(
                    event.first,
                    event.rows,
                    (event.filters['appId'] as FilterMetadata)?.value ?? null,
                    (event.filters['subnetId'] as FilterMetadata)?.value ?? null,
                    (event.filters['dhcpVersion'] as FilterMetadata)?.value ?? null,
                    (event.filters['text'] as FilterMetadata)?.value || null
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
        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    map((f) => {
                        return { ...f, value: f.value ?? null }
                    }),
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        // this.zone.run(() =>
                        this.router.navigate(
                            [],
                            { queryParams: tableFiltersToQueryParams(this.table) }
                            // )
                        )
                    })
                )
                .subscribe()
        )
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

    protected readonly tableHasFilter = tableHasFilter

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
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
    }
}
