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
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Subnets' }]

    @ViewChild('subnetsTable') subnetsTable: Table

    // subnets
    subnets: any[]
    totalSubnets = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    queryParams = {
        text: null,
        dhcpVersion: null,
        appId: null,
    }

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
            { label: 'any', value: null, id: 'any-menu' },
            { label: 'DHCPv4', value: '4', id: 'dhcpv4-menu' },
            { label: 'DHCPv6', value: '6', id: 'dhcpv6-menu' },
        ]

        this.settingSvc.getSettings().subscribe((data) => {
            this.grafanaUrl = data['grafana_url']
        })

        // handle initial query params
        const ssParams = this.route.snapshot.queryParamMap
        let text = ''
        if (ssParams.get('text')) {
            text += ' ' + ssParams.get('text')
        }
        if (ssParams.get('appId')) {
            text += ' appId:' + ssParams.get('appId')
        }
        this.filterText = text.trim()
        this.updateOurQueryParams(ssParams)

        // subscribe to subsequent changes to query params
        this.route.queryParamMap.subscribe((params) => {
            this.updateOurQueryParams(params)
            let event = { first: 0, rows: 10 }
            if (this.subnetsTable) {
                event = this.subnetsTable.createLazyLoadMetadata()
            }
            this.loadSubnets(event)
        })
    }

    updateOurQueryParams(params) {
        if (['4', '6'].includes(params.get('dhcpVersion'))) {
            this.queryParams.dhcpVersion = params.get('dhcpVersion')
        }
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
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
            .getSubnets(event.first, event.rows, params.appId, params.dhcpVersion, params.text)
            .subscribe((data) => {
                this.subnets = data.items
                this.totalSubnets = data.total
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
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId'], null)
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
