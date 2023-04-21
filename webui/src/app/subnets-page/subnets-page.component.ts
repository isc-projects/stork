import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { getGrafanaUrl, extractKeyValsAndPrepareQueryParams, getGrafanaSubnetTooltip, getErrorMessage } from '../utils'
import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    SubnetWithUniquePools,
    extractUniqueSubnetPools,
} from '../subnets'
import { SettingService } from '../setting.service'
import { Subscription } from 'rxjs'
import { map } from 'rxjs/operators'
import { Subnet } from '../backend'
import { MessageService } from 'primeng/api'

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
    queryParams = {
        text: null,
        dhcpVersion: null,
        appId: null,
        localSubnetId: null,
    }

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
        let text = ''
        if (ssParams.get('text')) {
            text += ' ' + ssParams.get('text')
        }
        if (ssParams.get('appId')) {
            text += ' appId:' + ssParams.get('appId')
        }
        if (ssParams.get('subnetId')) {
            text += ' subnetId:' + ssParams.get('subnetId')
        }
        this.filterText = text.trim()
        this.updateOurQueryParams(ssParams)

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
    }

    // ToDo: Silent error catching
    updateOurQueryParams(params) {
        if (['4', '6'].includes(params.get('dhcpVersion'))) {
            this.queryParams.dhcpVersion = params.get('dhcpVersion')
        }
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
        this.queryParams.localSubnetId = params.get('subnetId')
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
            .getSubnets(event.first, event.rows, params.appId, params.localSubnetId, params.dhcpVersion, params.text)
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
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId', 'subnetId'], null)
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
