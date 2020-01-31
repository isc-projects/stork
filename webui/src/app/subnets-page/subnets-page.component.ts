import { Component, OnInit } from '@angular/core'

import { DHCPService } from '../backend/api/api'

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
        this.dhcpVersions = [
            { name: 'any', value: '0' },
            { name: 'DHCPv4', value: '4' },
            { name: 'DHCPv6', value: '6' },
        ]
    }

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

    filterByDhcpVersion(subnetsTable) {
        subnetsTable.filter(this.selectedDhcpVersion.value, 'dhcpVersion', 'equals')
    }

    keyDownFilterText(subnetsTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            subnetsTable.filter(this.filterText, 'text', 'equals')
        }
    }
}
