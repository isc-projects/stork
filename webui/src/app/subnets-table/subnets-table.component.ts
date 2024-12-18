import { Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { DHCPService, Subnet } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Location } from '@angular/common'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage, getGrafanaSubnetTooltip, getGrafanaUrl } from '../utils'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SubnetWithUniquePools,
    extractUniqueSubnetPools,
} from '../subnets'
import { map } from 'rxjs/operators'

/**
 * Specifies the filter parameters for fetching subnets that may be specified
 * either in the URL query parameters or programmatically.
 */
export interface SubnetsFilter {
    text?: string
    appId?: number
    subnetId?: number
    dhcpVersion?: 4 | 6
}

@Component({
    selector: 'app-subnets-table',
    templateUrl: './subnets-table.component.html',
    styleUrl: './subnets-table.component.sass',
})
export class SubnetsTableComponent
    extends PrefilteredTable<SubnetsFilter, SubnetWithUniquePools>
    implements OnInit, OnDestroy
{
    /**
     * Array of all numeric keys that are supported when filtering subnets via URL queryParams.
     * Note that it doesn't have to contain subnets prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof SubnetsFilter)[] = ['dhcpVersion']

    /**
     * Array of all boolean keys that are supported when filtering subnets via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof SubnetsFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter subnets.
     */
    filterNumericKeys: (keyof SubnetsFilter)[] = ['appId', 'subnetId', 'dhcpVersion']

    /**
     * Array of all boolean keys that can be used to filter subnets.
     */
    filterBooleanKeys: (keyof SubnetsFilter)[] = []

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'subnets-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof SubnetsFilter = 'appId'

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values, e.g. dhcpVersion=4|6.
     */
    filterValidators = [{ filterKey: 'dhcpVersion', allowedValues: [4, 6] }]

    /**
     * PrimeNG table instance.
     */
    @ViewChild('subnetsTable') table: Table

    /**
     * URL to grafana.
     */
    @Input() grafanaUrl: string

    /**
     * Indicates if the data is being fetched from the server.
     */
    @Input() dataLoading: boolean = false

    constructor(
        private route: ActivatedRoute,
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private location: Location
    ) {
        super(route, location)
    }

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
                    this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                    this.getTableFilterValue('subnetId', event.filters),
                    this.getTableFilterValue('dhcpVersion', event.filters),
                    this.getTableFilterValue('text', event.filters)
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
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
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
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }

    /**
     * Builds a tooltip explaining what the link is for.
     * @param subnet an identifier of the subnet
     * @param machine an identifier of the machine the subnet is configured on
     */
    getGrafanaTooltip(subnet: number, machine: string) {
        return getGrafanaSubnetTooltip(subnet, machine)
    }
}
