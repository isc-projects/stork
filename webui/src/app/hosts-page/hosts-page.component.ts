import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { extractKeyValsAndPrepareQueryParams } from '../utils'

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
    @ViewChild('hostsTable') hostsTable: Table

    // hosts
    hosts: any[]
    totalHosts = 0

    // filters
    filterText = ''
    queryParams = {
        text: null,
        appId: null,
    }

    constructor(private route: ActivatedRoute, private router: Router, private dhcpApi: DHCPService) {}

    ngOnInit() {
        // handle initial query params
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
            if (this.hostsTable) {
                event = this.hostsTable.createLazyLoadMetadata()
            }
            this.loadHosts(event)
        })
    }

    updateOurQueryParams(params) {
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
    }

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     *              number of rows to be returned and a text for hosts filtering.
     */
    loadHosts(event) {
        const params = this.queryParams

        this.dhcpApi.getHosts(event.first, event.rows, params.appId, null, params.text).subscribe((data) => {
            this.hosts = data.items
            this.totalHosts = data.total
        })
    }

    /**
     * Filters the list of hosts by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyUpFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId'])
            this.router.navigate(['/dhcp/hosts'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Returns tooltip explaining where the server has the given host
     * reservation specified, i.e. in the configuration file or a database.
     *
     * @param dataSource data source provided as a string.
     * @returns The tooltip text.
     */
    hostDataSourceTooltip(dataSource): string {
        switch (dataSource) {
            case 'config':
                return 'The server has this host specified in the configuration file.'
            case 'api':
                return 'The server has this host specified in the host database.'
            default:
                break
        }
        return ''
    }
}
