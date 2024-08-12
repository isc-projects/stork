import { Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SharedNetworkWithUniquePools,
} from '../subnets'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { DHCPService, SharedNetwork } from '../backend'
import { MessageService } from 'primeng/api'
import { Location } from '@angular/common'
import { lastValueFrom } from 'rxjs'
import { map } from 'rxjs/operators'
import { getErrorMessage } from '../utils'

/**
 * Specifies the filter parameters for fetching shared networks that may be specified
 * either in the URL query parameters or programmatically.
 */
export interface SharedNetworksFilter {
    text?: string
    appId?: number
    dhcpVersion?: 4 | 6
}

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-table',
    templateUrl: './shared-networks-table.component.html',
    styleUrl: './shared-networks-table.component.sass',
})
export class SharedNetworksTableComponent
    extends PrefilteredTable<SharedNetworksFilter, SharedNetworkWithUniquePools>
    implements OnInit, OnDestroy
{
    /**
     * Array of all numeric keys that are supported when filtering shared networks via URL queryParams.
     * Note that it doesn't have to contain shared networks prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof SharedNetworksFilter)[] = ['dhcpVersion']

    /**
     * Array of all boolean keys that are supported when filtering shared networks via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof SharedNetworksFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter shared networks.
     */
    filterNumericKeys: (keyof SharedNetworksFilter)[] = ['appId', 'dhcpVersion']

    /**
     * Array of all boolean keys that can be used to filter shared networks.
     */
    filterBooleanKeys: (keyof SharedNetworksFilter)[] = []

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'networks-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof SharedNetworksFilter = 'appId'

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values, e.g. dhcpVersion=4|6.
     */
    filterValidators = [{ filterKey: 'dhcpVersion', allowedValues: [4, 6] }]

    /**
     * PrimeNG table instance.
     */
    @ViewChild('networksTable') table: Table

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

        lastValueFrom(
            this.dhcpApi
                .getSharedNetworks(
                    event.first,
                    event.rows,
                    this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                    this.getTableFilterValue('dhcpVersion', event.filters),
                    this.getTableFilterValue('text', event.filters)
                )
                .pipe(
                    map((sharedNetworks) => {
                        parseSubnetsStatisticValues(sharedNetworks.items)
                        return sharedNetworks
                    })
                )
        )
            .then((data) => {
                this.dataCollection = data.items || []
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
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
    }

    /**
     * Returns true if at least one of the shared networks contains at least
     * one IPv6 subnet
     */
    get isAnyIPv6SubnetVisible(): boolean {
        return !!this.dataCollection?.some((n) => n.subnets.some((s) => s.subnet.includes(':')))
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
        return network.stats?.['total-pds']
    }

    /**
     * Get the number of delegated prefixes in the network.
     */
    getAssignedDelegatedPrefixes(network: SharedNetwork) {
        return network.stats?.['assigned-pds']
    }

    /**
     * Returns a list of applications maintaining a given shared network.
     * The list doesn't contain duplicates.
     *
     * @param net Shared network
     * @returns List of the applications (only ID and app name)
     */
    getApps(net: SharedNetwork) {
        const apps = []
        const appIds = {}

        if (net.localSharedNetworks) {
            net.localSharedNetworks.forEach((lsn) => {
                if (!appIds.hasOwnProperty(lsn.appId)) {
                    apps.push({ id: lsn.appId, name: lsn.appName })
                    appIds[lsn.appId] = true
                }
            })
        }

        return apps
    }
}
