import { Component, OnInit } from '@angular/core'

import { DHCPService } from '../backend/api/api'
import { humanCount } from '../utils'
import { getTotalAddresses, getAssignedAddresses } from '../subnets'

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit {
    // subnets
    subnets: any[]
    totalSubnets = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    selectedDhcpVersion: any

    constructor(private dhcpApi: DHCPService) {}

    ngOnInit() {
        // prepare list of DHCP versions, this is used in subnets filtering
        this.dhcpVersions = [
            { name: 'any', value: '0' },
            { name: 'DHCPv4', value: '4' },
            { name: 'DHCPv6', value: '6' },
        ]
    }

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for subnets filtering.
     */
    loadSubnets(event) {
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        let dhcpVersion
        if (event.filters.dhcpVersion) {
            dhcpVersion = event.filters.dhcpVersion.value
        }

        this.dhcpApi.getSubnets(event.first, event.rows, null, dhcpVersion, text).subscribe(data => {
            this.subnets = data.items
            this.totalSubnets = data.total
        })
    }

    /**
     * Filters list of subnets by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion(subnetsTable) {
        subnetsTable.filter(this.selectedDhcpVersion.value, 'dhcpVersion', 'equals')
    }

    /**
     * Filters list of subnets by text. Filtering is realized server-side.
     */
    keyDownFilterText(subnetsTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            subnetsTable.filter(this.filterText, 'text', 'equals')
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
}
