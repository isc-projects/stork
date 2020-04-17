import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { humanCount, getGrafanaUrl, extractKeyValsAndPrepareQueryParams } from '../utils'
import { getTotalAddresses, getAssignedAddresses } from '../subnets'
import { SettingService } from '../setting.service'

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit {
    @ViewChild('subnetsTable', undefined) subnetsTable: Table

    // subnets
    subnets: any[]
    totalSubnets = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    selectedDhcpVersion: any

    grafanaUrl: string

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private dhcpApi: DHCPService,
        private settingSvc: SettingService
    ) {}

    ngOnInit() {
        // prepare list of DHCP versions, this is used in subnets filtering
        this.dhcpVersions = [
            { name: 'any', value: null },
            { name: 'DHCPv4', value: '4' },
            { name: 'DHCPv6', value: '6' },
        ]

        this.settingSvc.getSettings().subscribe(data => {
            this.grafanaUrl = data['grafana_url']
        })

        // handle initial query params
        const params = this.route.snapshot.queryParams
        if (params.dhcpVersion === '4') {
            this.selectedDhcpVersion = this.dhcpVersions[1]
        } else if (params.dhcpVersion === '6') {
            this.selectedDhcpVersion = this.dhcpVersions[2]
        }
        let text = ''
        if (params.appId) {
            text += ' appId=' + params.appId
        }
        this.filterText = text.trim()

        // subscribe to subsequent changes to query params
        this.route.queryParamMap.subscribe(data => {
            const event = this.subnetsTable.createLazyLoadMetadata()
            this.loadSubnets(event)
        })
    }

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for subnets filtering.
     */
    loadSubnets(event) {
        const params = this.route.snapshot.queryParams
        const appId = params.appId
        const dhcpVersion = params.dhcpVersion
        const text = params.text

        this.dhcpApi.getSubnets(event.first, event.rows, appId, dhcpVersion, text).subscribe(data => {
            this.subnets = data.items
            this.totalSubnets = data.total
        })
    }

    /**
     * Filters list of subnets by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/subnets'], {
            queryParams: { dhcpVersion: this.selectedDhcpVersion.value },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters list of subnets by text. The text may contain key=val pairs what
     * allows filtering by other keys. Filtering is realized server-side.
     */
    keyupFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appid'])

            this.router.navigate(['/dhcp/subnets'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Prepare count for presenting in a column that it is easy to grasp by humans.
     */
    humanCount(count) {
        if (isNaN(count)) {
            return count
        }
        if (Math.abs(count) < 1000000) {
            return count.toLocaleString('en-US')
        }
        return humanCount(count)
    }

    /**
     * Prepare count for presenting in tooltip by adding ',' separator to big numbers, eg. 1,243,342.
     */
    tooltipCount(count) {
        if (count === '?') {
            return 'not data collected yet'
        }
        return count.toLocaleString('en-US')
    }

    /**
     * Get total number of addresses in a subnet.
     */
    getTotalAddresses(subnet) {
        if (subnet.localSubnets[0].stats) {
            return getTotalAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Get assigned number of addresses in a subnet.
     */
    getAssignedAddresses(subnet) {
        if (subnet.localSubnets[0].stats) {
            return getAssignedAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }
}
