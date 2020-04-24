import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { humanCount, extractKeyValsAndPrepareQueryParams } from '../utils'
import { getTotalAddresses, getAssignedAddresses } from '../subnets'

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-page',
    templateUrl: './shared-networks-page.component.html',
    styleUrls: ['./shared-networks-page.component.sass'],
})
export class SharedNetworksPageComponent implements OnInit {
    @ViewChild('networksTable') networksTable: Table

    // networks
    networks: any[]
    totalNetworks = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    queryParams = {
        text: null,
        dhcpVersion: null,
        appId: null,
    }

    constructor(private route: ActivatedRoute, private router: Router, private dhcpApi: DHCPService) {}

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
            text += ' appId=' + ssParams.get('appId')
        }
        this.filterText = text.trim()
        this.updateOurQueryParams(ssParams)

        // subscribe to subsequent changes to query params
        this.route.queryParamMap.subscribe((params) => {
            this.updateOurQueryParams(params)
            let event = { first: 0, rows: 10 }
            if (this.networksTable) {
                event = this.networksTable.createLazyLoadMetadata()
            }
            this.loadNetworks(event)
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
     * Loads networks from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for networks filtering.
     */
    loadNetworks(event) {
        const params = this.queryParams
        this.dhcpApi
            .getSharedNetworks(event.first, event.rows, params.appId, params.dhcpVersion, params.text)
            .subscribe((data) => {
                this.networks = data.items
                this.totalNetworks = data.total
            })
    }

    /**
     * Filters list of networks by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/shared-networks'], {
            queryParams: { dhcpVersion: this.queryParams.dhcpVersion },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters list of networks by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyupFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId'])

            this.router.navigate(['/dhcp/shared-networks'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Get total of addresses in the network by summing up all subnets.
     */
    getTotalAddresses(network) {
        let total = 0
        for (const sn of network.subnets) {
            if (sn.localSubnets[0].stats) {
                total += getTotalAddresses(sn)
            }
        }
        return total
    }

    /**
     * Get assigned of addresses in the network by summing up all subnets.
     */
    getAssignedAddresses(network) {
        let total = 0
        for (const sn of network.subnets) {
            if (sn.localSubnets[0].stats) {
                total += getAssignedAddresses(sn)
            }
        }
        return total
    }

    /**
     * Prepare count for presenting in tooltip by adding ',' separator to big numbers, eg. 1,243,342.
     */
    tooltipCount(count) {
        return count.toLocaleString('en-US')
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

    getApps(net) {
        const apps = []
        const appIds = {}

        for (const sn of net.subnets) {
            for (const lsn of sn.localSubnets) {
                if (!appIds.hasOwnProperty(lsn.appId)) {
                    apps.push({ id: lsn.appId, machineAddress: lsn.machineAddress })
                    appIds[lsn.appId] = true
                }
            }
        }

        return apps
    }
}
