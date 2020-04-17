import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { humanCount } from '../utils'
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
    @ViewChild('networksTable', undefined) networksTable: Table

    // networks
    networks: any[]
    totalNetworks = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    selectedDhcpVersion: any

    constructor(private route: ActivatedRoute, private router: Router, private dhcpApi: DHCPService) {}

    ngOnInit() {
        // prepare list of DHCP versions, this is used in networks filtering
        this.dhcpVersions = [
            { name: 'any', value: null },
            { name: 'DHCPv4', value: '4' },
            { name: 'DHCPv6', value: '6' },
        ]

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
            const event = this.networksTable.createLazyLoadMetadata()
            this.loadNetworks(event)
        })
    }

    /**
     * Loads networks from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for networks filtering.
     */
    loadNetworks(event) {
        const params = this.route.snapshot.queryParams
        const appId = params.appId
        const dhcpVersion = params.dhcpVersion
        const text = params.text

        this.dhcpApi.getSharedNetworks(event.first, event.rows, appId, dhcpVersion, text).subscribe(data => {
            this.networks = data.items
            this.totalNetworks = data.total
        })
    }

    /**
     * Filters list of networks by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/shared-networks'], {
            queryParams: { dhcpVersion: this.selectedDhcpVersion.value },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters list of networks by text. Filtering is realized server-side.
     */
    keyupFilterText(event) {
        console.info(event)
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            let text = this.filterText

            // find all occurences key=val in the text
            const re = /(\w+)=(\w*)/g
            const matches = []
            let match = re.exec(text)
            while (match !== null) {
                matches.push(match)
                match = re.exec(text)
            }

            const queryParams = {
                appId: null,
                text: null,
            }
            for (const m of matches) {
                text = text.replace(m[0], '')
                if (m[1].toLowerCase() === 'appid') {
                    queryParams.appId = m[2]
                }
            }
            text = text.trim()
            if (text) {
                queryParams.text = text
            }

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
