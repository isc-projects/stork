import { Component, OnInit } from '@angular/core'

import { DHCPService } from '../backend/api/api'

/**
 * This component implemnents a page which displays hosts along with
 * their DHCP identifiers and IP reservations. The list of hosts is
 * paged and can be filtered by a reserved IP address. The list
 * contains host reservations for all subnets and in the future it
 * will also contain global reservations, i.e. those that are not
 * associated with any particular subnet.
 */
@Component({
    selector: 'app-hosts-page',
    templateUrl: './hosts-page.component.html',
    styleUrls: ['./hosts-page.component.sass'],
})
export class HostsPageComponent implements OnInit {
    // hosts
    hosts: any[]
    totalHosts = 0

    // filters
    filterText = ''

    constructor(private dhcpApi: DHCPService) {}

    ngOnInit() {}

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     *              number of rows to be returned and a text for hosts filtering.
     */
    loadHosts(event) {
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        this.dhcpApi.getHosts(event.first, event.rows, null, text).subscribe(data => {
            this.hosts = data.items
            this.totalHosts = data.total
        })
    }

    /**
     * Filters the list of hosts by text. Filtering is performed on the server
     */
    keyDownFilterText(hostsTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            hostsTable.filter(this.filterText, 'text', 'equals')
        }
    }
}
