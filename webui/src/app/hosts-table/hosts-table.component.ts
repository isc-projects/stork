import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { DHCPService, Host, LocalHost } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { Location } from '@angular/common'
import { MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { hasDifferentLocalHostData } from '../hosts'
import { lastValueFrom } from 'rxjs'

/**
 * Specifies the filter parameters for fetching hosts that may be specified
 * either in the URL query parameters or programmatically.
 */
export interface HostsFilter {
    text?: string
    appId?: number
    subnetId?: number
    keaSubnetId?: number
    isGlobal?: boolean
    conflict?: boolean
}

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
})
export class HostsTableComponent extends PrefilteredTable<HostsFilter, Host> implements OnInit, OnDestroy {
    /**
     * Array of all numeric keys that are supported when filtering hosts via URL queryParams.
     * Note that it doesn't have to contain hosts prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof HostsFilter)[] = []

    /**
     * Array of all boolean keys that are supported when filtering hosts via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof HostsFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter hosts.
     */
    filterNumericKeys: (keyof HostsFilter)[] = ['appId', 'subnetId', 'keaSubnetId']

    /**
     * Array of all boolean keys that can be used to filter hosts.
     */
    filterBooleanKeys: (keyof HostsFilter)[] = ['isGlobal', 'conflict']

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'hosts-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof HostsFilter = 'appId'

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values.
     */
    filterValidators = []

    /**
     * PrimeNG table instance.
     */
    @ViewChild('hostsTable') table: Table

    constructor(
        private route: ActivatedRoute,
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private location: Location
    ) {
        super(route, location)
    }

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
                this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                this.getTableFilterValue('subnetId', event.filters),
                this.getTableFilterValue('keaSubnetId', event.filters),
                this.getTableFilterValue('text', event.filters),
                this.getTableFilterValue('isGlobal', event.filters),
                this.getTableFilterValue('conflict', event.filters)
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
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
    }
}
