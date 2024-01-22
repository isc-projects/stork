import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute, ParamMap } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { extractKeyValsAndPrepareQueryParams, getErrorMessage } from '../utils'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    parseSubnetStatisticValues,
    extractUniqueSharedNetworkPools,
    SharedNetworkWithUniquePools,
} from '../subnets'
import { Subscription, lastValueFrom } from 'rxjs'
import { map } from 'rxjs/operators'
import { SharedNetwork } from '../backend'
import { MenuItem, MessageService } from 'primeng/api'

/**
 * Specifies the filter parameters for fetching shared networks that may be
 * specified in the URL query parameters.
 */
interface QueryParamsFilter {
    text: string
    dhcpVersion: '4' | '6'
    appId: string
}

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-page',
    templateUrl: './shared-networks-page.component.html',
    styleUrls: ['./shared-networks-page.component.sass'],
})
export class SharedNetworksPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Shared Networks' }]

    @ViewChild('networksTable') networksTable: Table

    // networks
    networks: SharedNetworkWithUniquePools[] = []
    totalNetworks = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    queryParams: QueryParamsFilter = {
        text: null,
        dhcpVersion: null,
        appId: null,
    }

    // Tab menu

    /**
     * Array of tab menu items with shared network information.
     *
     * The first tab is always present and displays the shared networks list.
     *
     * Note: we cannot use the URL with no segment for the list tab. It causes
     * the first tab to be always marked active. The Tab Menu has a built-in
     * feature to highlight items based on the current route. It seems that it
     * matches by a prefix instead of an exact value (the "/foo/bar" URL
     * matches the menu item with the "/foo" URL).
     */
    tabs: MenuItem[] = [{ label: 'Shared Networks', routerLink: '/dhcp/shared-networks/all' }]

    /**
     * Selected tab menu index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Indicates if the component is loading data from the server.
     */
    loading: boolean = false

    /**
     * Holds the information about specific shared networks presented in the tabs.
     *
     * The entry corresponding to shared networks list is not related to any specific
     * shared network. Its ID is 0.
     */
    openedSharedNetworks: SharedNetworkWithUniquePools[] = [{ id: 0 }]

    /**
     * Constructor.
     *
     * @param route activated route.
     * @param messageService message service.
     * @param router router.
     * @param dhcpApi a service for communication with the server.
     */
    constructor(
        private route: ActivatedRoute,
        private messageService: MessageService,
        private router: Router,
        private dhcpApi: DHCPService
    ) {}

    /**
     * A component lifecycle hook invoked when the component instance is destroyed.
     *
     * It unsubscribes from all subscriptions.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * A component lifecycle hook invoked when the component is initialized.
     */
    ngOnInit() {
        // prepare list of DHCP versions, this is used in networks filtering
        this.dhcpVersions = [
            { label: 'any', value: null },
            { label: 'DHCPv4', value: '4' },
            { label: 'DHCPv6', value: '6' },
        ]

        const ssParams = this.route.snapshot.queryParamMap
        let text = ''
        if (ssParams.get('text')) {
            text += ' ' + ssParams.get('text')
        }
        if (ssParams.get('appId')) {
            text += ' appId:' + ssParams.get('appId')
        }
        this.filterText = text.trim()

        // subscribe to subsequent changes to query params
        this.subscriptions.add(
            this.route.queryParamMap.subscribe(
                (params) => {
                    this.updateOurQueryParams(params)
                    let event = { first: 0, rows: 10 }
                    if (this.networksTable) {
                        // When filtering is applied, go by default to first page after filtering.
                        this.networksTable.first = 0
                        event = this.networksTable.createLazyLoadMetadata()
                    }
                    this.loadNetworks(event)
                },
                // ToDo: Silent error catching
                (error) => {
                    console.log(error)
                }
            )
        )

        // Subscribe to the shared network id changes, e.g. from /dhcp/shared-networks/all to
        // /dhcp/shared-networks/1. These changes are triggered by switching between the
        // tabs.
        this.subscriptions.add(
            this.route.paramMap.subscribe(
                (params) => {
                    // Get shared network id.
                    const id = params.get('id')
                    let numericId = parseInt(id, 10)
                    if (Number.isNaN(numericId)) {
                        numericId = 0
                    }
                    this.openTabBySharedNetworkId(numericId)
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot process URL segment parameters',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )
    }

    /**
     * Update various component's query parameters from the URL
     * query parameters.
     *
     * @param params query parameters.
     */
    updateOurQueryParams(params: ParamMap) {
        if (['4', '6'].includes(params.get('dhcpVersion'))) {
            this.queryParams.dhcpVersion = params.get('dhcpVersion') as '4' | '6'
        }
        this.queryParams.text = params.get('text') || null
        this.queryParams.appId = params.get('appId') || null
    }

    /**
     * Loads shared networks from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for networks filtering.
     */
    loadNetworks(event) {
        if (this.loading) {
            return
        }
        this.loading = true

        const params = this.queryParams
        lastValueFrom(
            this.dhcpApi
                .getSharedNetworks(
                    event.first,
                    event.rows,
                    Number(params.appId) || null,
                    Number(params.dhcpVersion) || null,
                    params.text
                )
                .pipe(
                    map((sharedNetworks) => {
                        parseSubnetsStatisticValues(sharedNetworks.items)
                        return sharedNetworks
                    })
                )
        )
            .then((data) => {
                this.networks = data.items
                this.totalNetworks = data.total ?? 0
            })
            .catch((error) => {
                // ToDo: Silent error catching. We should display a message to the user.
                console.log(error)
            })
            .finally(() => {
                this.loading = false
            })
    }

    /**
     * Filters the list of shared networks by DHCP versions. Filtering is performed
     * by the server.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/shared-networks'], {
            queryParams: { dhcpVersion: this.queryParams.dhcpVersion },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters the list of shared networks by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is performed by
     * the server.
     */
    keyupFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams<QueryParamsFilter>(this.filterText, ['appId'], null)

            this.router.navigate(['/dhcp/shared-networks'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
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

        for (const sn of net.subnets) {
            for (const lsn of sn.localSubnets) {
                if (!appIds.hasOwnProperty(lsn.appId)) {
                    apps.push({ id: lsn.appId, name: lsn.appName })
                    appIds[lsn.appId] = true
                }
            }
        }

        return apps
    }

    /**
     * Returns true if at least one of the shared networks contains at least
     * one IPv6 subnet
     */
    get isAnyIPv6SubnetVisible(): boolean {
        return !!this.networks?.some((n) => n.subnets.some((s) => s.subnet.includes(':')))
    }

    /**
     * Open a shared network tab.
     *
     * If the tab already exists, switch to it without fetching the data.
     * Otherwise, fetch the shared network information from the API and
     * create a new tab.
     *
     * @param sharedNetworkId Shared network ID or a NaN for subnet list.
     */
    openTabBySharedNetworkId(sharedNetworkId: number) {
        const tabIndex = this.openedSharedNetworks.map((t) => t.id).indexOf(sharedNetworkId)
        if (tabIndex < 0) {
            this.createTab(sharedNetworkId).then(() => {
                this.switchToTab(this.openedSharedNetworks.length - 1)
            })
        } else {
            this.switchToTab(tabIndex)
        }
    }

    /**
     * Close a menu tab by index.
     *
     * @param index Tab index.
     * @param event Event triggered upon tab closing.
     */
    closeTabByIndex(index: number, event?: Event) {
        if (index == 0) {
            return
        }

        this.openedSharedNetworks.splice(index, 1)
        this.tabs.splice(index, 1)

        if (this.activeTabIndex === index) {
            // Closing currently selected tab. Switch to previous tab.
            this.switchToTab(index - 1)
            this.router.navigate([this.tabs[index - 1].routerLink])
        } else if (this.activeTabIndex > index) {
            // Sitting on the later tab then the one closed. We don't need
            // to switch, but we have to adjust the active tab index.
            this.activeTabIndex--
        }

        if (event) {
            event.preventDefault()
        }
    }

    /**
     * Create a new tab for a given shared network ID.
     *
     * It fetches the shared network information from the API.
     *
     * @param sharedNetworkId Shared network ID.
     */
    private createTab(sharedNetworkId: number): Promise<void> {
        return (
            lastValueFrom(
                // Fetch data from API.
                this.dhcpApi.getSharedNetwork(sharedNetworkId).pipe(
                    map((sharedNetwork) => {
                        if (sharedNetwork) {
                            parseSubnetStatisticValues(sharedNetwork)
                        }
                        return sharedNetwork
                    })
                )
            )
                // Execute and use.
                .then((data) => {
                    if (data) {
                        const networks = extractUniqueSharedNetworkPools([data])
                        this.appendTab(networks[0])
                    }
                })
                .catch((error) => {
                    const msg = getErrorMessage(error)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot get shared network',
                        detail: `Error getting shared network with ID ${sharedNetworkId}: ${msg}`,
                        life: 10000,
                    })
                })
        )
    }

    /**
     * Append a new tab to the list of tabs.
     *
     * @param sharedNetwork Shared network data.
     */
    private appendTab(sharedNetwork: SharedNetwork) {
        this.openedSharedNetworks.push(sharedNetwork)
        this.tabs.push({
            label: sharedNetwork.name,
            routerLink: `/dhcp/shared-networks/${sharedNetwork.id}`,
        })
    }

    /**
     * Switch to tab identified by an index.
     *
     * @param index Tab index.
     */
    private switchToTab(index: number) {
        if (this.activeTabIndex === index) {
            return
        }
        this.activeTabIndex = index
    }
}
