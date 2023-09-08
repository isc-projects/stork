import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute, ParamMap } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { getGrafanaUrl, extractKeyValsAndPrepareQueryParams, getGrafanaSubnetTooltip, getErrorMessage } from '../utils'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SubnetWithUniquePools,
    extractUniqueSubnetPools,
    parseSubnetStatisticValues,
} from '../subnets'
import { SettingService } from '../setting.service'
import { Subscription, concat, of } from 'rxjs'
import { filter, map, take } from 'rxjs/operators'
import { Subnet } from '../backend'
import { MenuItem, MessageService } from 'primeng/api'

interface QueryParamsFilter {
    text: string
    dhcpVersion: '4' | '6'
    appId: string
    subnetId: string
}

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Subnets' }]

    @ViewChild('subnetsTable') subnetsTable: Table

    // subnets
    subnets: SubnetWithUniquePools[] = []
    totalSubnets = 0

    // filters
    filterText = ''
    queryParams: QueryParamsFilter = {
        text: null,
        dhcpVersion: null,
        appId: null,
        subnetId: null,
    }

    // Tab menu

    /**
     * Array of tab menu items with subnet information.
     *
     * The first tab is always present and displays the subnet list.
     *
     * Note: we cannot use the URL with no segment for the list tab. It causes
     * the first tab to be always marked active. The Tab Menu has a built-in
     * feature to highlight items based on the current route. It seems that it
     * matches by a prefix instead of an exact value (the "/foo/bar" URL
     * matches the menu item with the "/foo" URL).
     */
    tabs: MenuItem[] = [{ label: 'Subnets', routerLink: '/dhcp/subnets/all' }]

    /**
     * Selected tab menu index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Holds the information about specific subnets presented in the tabs.
     *
     * The entry corresponding to subnets list is not related to any specific
     * subnet. Its ID is 0.
     */
    openedSubnets: Subnet[] = [{ id: 0 }]

    grafanaUrl: string

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private dhcpApi: DHCPService,
        private settingSvc: SettingService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    ngOnInit() {
        // ToDo: Silent error catching
        this.subscriptions.add(
            this.settingSvc.getSettings().subscribe(
                (data) => {
                    this.grafanaUrl = data['grafana_url']
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot fetch server settings',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )

        // handle initial query params
        const ssParams = this.route.snapshot.queryParamMap
        this.updateFilterText(ssParams)

        // subscribe to subsequent changes to query params
        this.subscriptions.add(
            this.route.queryParamMap.subscribe(
                (params) => {
                    this.updateOurQueryParams(params)
                    let event = { first: 0, rows: 10 }
                    if (this.subnetsTable) {
                        event = this.subnetsTable.createLazyLoadMetadata()
                    }
                    this.loadSubnets(event)
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot process URL query parameters',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )

        // Subscribe to the subnet id changes, e.g. from /dhcp/subnets/all to
        // /dhcp/subnets/1. These changes are triggered by switching between the
        // tabs.
        this.subscriptions.add(
            this.route.paramMap.subscribe(
                (params) => {
                    // Get subnet id.
                    const id = params.get('id')
                    let numericId = parseInt(id, 10)
                    if (Number.isNaN(numericId)) {
                        numericId = 0
                    }
                    this.openTabBySubnetId(numericId)
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
     * Update different component's query parameters from the URL
     * query parameters.
     *
     * @param params query parameters.
     */
    private updateOurQueryParams(params: ParamMap) {
        if (['4', '6'].includes(params.get('dhcpVersion'))) {
            this.queryParams.dhcpVersion = params.get('dhcpVersion') as '4' | '6'
        }
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
        this.queryParams.subnetId = params.get('subnetId')
    }

    /**
     * Set the filter text value using the URL query parameters.
     *
     * @param params query parameters.
     */
    private updateFilterText(params: ParamMap) {
        let text = ''
        if (params.has('appId')) {
            text += ' appId:' + params.get('appId')
        }
        if (params.has('subnetId')) {
            text += ' subnetId:' + params.get('subnetId')
        }
        if (params.has('text')) {
            text += ' ' + params.get('text')
        }
        this.filterText = text.trim()
    }

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for subnets filtering.
     */
    loadSubnets(event) {
        const params = this.queryParams

        this.dhcpApi
            .getSubnets(
                event.first, event.rows,
                Number(params.appId) || null,
                Number(params.subnetId) || null,
                Number(params.dhcpVersion) || null,
                params.text
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
            .toPromise()
            .then((data) => {
                this.subnets = data.items ? extractUniqueSubnetPools(data.items) : null
                this.totalSubnets = data.total ?? 0
            })
            .catch((error) => {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot load subnets',
                    detail: getErrorMessage(error),
                })
            })
    }

    /**
     * Filters list of subnets by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/subnets'], {
            queryParams: { dhcpVersion: this.queryParams.dhcpVersion },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters list of subnets by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyupFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams<QueryParamsFilter>(this.filterText, ['appId', 'subnetId'], null)
            this.router.navigate(['/dhcp/subnets'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Prepare count for presenting in tooltip by adding ',' separator to big numbers, eg. 1,243,342.
     */
    tooltipCount(count) {
        if (count === '?') {
            return 'No data collected yet'
        }
        return count.toLocaleString('en-US')
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
     * Get total number of addresses in a subnet.
     */
    getTotalAddresses(subnet: Subnet) {
        if (subnet.stats) {
            return getTotalAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Get assigned number of addresses in a subnet.
     */
    getAssignedAddresses(subnet: Subnet) {
        if (subnet.stats) {
            return getAssignedAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Get total number of delegated prefixes in a subnet.
     */
    getTotalDelegatedPrefixes(subnet: Subnet) {
        if (subnet.subnet.includes('.')) {
            return null
        }
        return subnet.stats?.['total-pds'] ?? '?'
    }

    /**
     * Get assigned number of delegated prefixes in a subnet.
     */
    getAssignedDelegatedPrefixes(subnet: Subnet) {
        if (subnet.subnet.includes('.')) {
            return null
        }
        return subnet.stats?.['assigned-pds'] ?? '?'
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }

    /**
     * Open a subnet tab.
     *
     * If the tab already exists, switch to it without fetching the data.
     * Otherwise, fetch the subnet information from the API and create a
     * new tab.
     *
     * @param subnetId Subnet ID or a NaN for subnet list.
     */
    openTabBySubnetId(subnetId: number) {
        const tabIndex = this.openedSubnets.map((t) => t.id).indexOf(subnetId)
        if (tabIndex < 0) {
            this.createTab(subnetId).then(() => {
                this.switchToTab(this.openedSubnets.length - 1)
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

        this.openedSubnets.splice(index, 1)
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
     * Create a new tab for a given subnet ID.
     *
     * It fetches the subnet information from the API.
     *
     * @param subnetId Subnet Id.
     */
    private createTab(subnetId: number): Promise<void> {
        return (
            concat(
                // Existing entry or undefined.
                of(this.subnets.filter((s) => s.id == subnetId)[0])
                    // Drop an undefined value if the entry was not found.
                    .pipe(filter((s) => !!s)),
                // Fetch data from API.
                this.dhcpApi.getSubnet(subnetId)
            )
                // Take 1 item (return existing entry if exist, otherwise fetch the API).
                .pipe(take(1))
                .pipe(
                    map((subnet) => {
                        if (subnet) {
                            parseSubnetStatisticValues(subnet)
                        }
                        return subnet
                    })
                )
                // Execute and use.
                .toPromise()
                .then((data) => {
                    this.appendTab(data)
                })
                .catch((error) => {
                    const msg = getErrorMessage(error)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot get subnet',
                        detail: `Error getting subnet with ID ${subnetId}: ${msg}`,
                        life: 10000,
                    })
                })
        )
    }

    /**
     * Append a new tab to the list of tabs.
     *
     * @param subnet Subnet data.
     */
    private appendTab(subnet: Subnet) {
        this.openedSubnets.push(subnet)
        this.tabs.push({
            label: subnet.subnet,
            routerLink: `/dhcp/subnets/${subnet.id}`,
        })
    }

    /**
     * Switch to tab identified by an index.
     *
     * @param index Tab index.
     */
    private switchToTab(index: number) {
        if (this.activeTabIndex == index) {
            return
        }
        this.activeTabIndex = index
    }

    /**
     * Returns true if the subnet list presents at least one IPv6 subnet.
     */
    get isAnyIPv6SubnetVisible(): boolean {
        return !!this.subnets?.some((s) => s.subnet.includes(':'))
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
}
