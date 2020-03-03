import { Component, OnInit } from '@angular/core'

import { DHCPService } from '../backend/api/api'
import { getSubnetUtilization } from '../utils'

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-page',
    templateUrl: './shared-networks-page.component.html',
    styleUrls: ['./shared-networks-page.component.sass'],
})
export class SharedNetworksPageComponent implements OnInit {
    // networks
    networks: any[]
    totalNetworks = 0

    // filters
    filterText = ''
    dhcpVersions: any[]
    selectedDhcpVersion: any

    constructor(private dhcpApi: DHCPService) {}

    ngOnInit() {
        // prepare list of DHCP versions, this is used in networks filtering
        this.dhcpVersions = [
            { name: 'any', value: '0' },
            { name: 'DHCPv4', value: '4' },
            { name: 'DHCPv6', value: '6' },
        ]
    }

    /**
     * Loads networks from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for networks filtering.
     */
    loadNetworks(event) {
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        let dhcpVersion
        if (event.filters.dhcpVersion) {
            dhcpVersion = event.filters.dhcpVersion.value
        }

        this.dhcpApi.getSharedNetworks(event.first, event.rows, null, dhcpVersion, text).subscribe(data => {
            this.networks = data.items
            this.totalNetworks = data.total
        })
    }

    /**
     * Filters list of networks by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion(networksTable) {
        networksTable.filter(this.selectedDhcpVersion.value, 'dhcpVersion', 'equals')
    }

    /**
     * Filters list of networks by text. Filtering is realized server-side.
     */
    keyDownFilterText(networksTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            networksTable.filter(this.filterText, 'text', 'equals')
        }
    }
}
