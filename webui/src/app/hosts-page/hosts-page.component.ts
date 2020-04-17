import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

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
    @ViewChild('hostsTable', undefined) hostsTable: Table

    // hosts
    hosts: any[]
    totalHosts = 0

    // filters
    filterText = ''

    constructor(private route: ActivatedRoute, private router: Router, private dhcpApi: DHCPService) {}

    ngOnInit() {
        // handle initial query params
        const params = this.route.snapshot.queryParams
        let text = ''
        if (params.appId) {
            text += ' appId=' + params.appId
        }
        this.filterText = text.trim()

        // subscribe to subsequent changes to query params
        this.route.queryParamMap.subscribe(data => {
            const event = this.hostsTable.createLazyLoadMetadata()
            this.loadHosts(event)
        })
    }

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     *              number of rows to be returned and a text for hosts filtering.
     */
    loadHosts(event) {
        const params = this.route.snapshot.queryParams
        const appId = params.appId
        const text = params.text

        this.dhcpApi.getHosts(event.first, event.rows, appId, null, text).subscribe(data => {
            this.hosts = data.items
            this.totalHosts = data.total
        })
    }

    /**
     * Filters the list of hosts by text. Filtering is performed on the server
     */
    keyUpFilterText(event) {
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

            this.router.navigate(['/dhcp/hosts'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }
}
