import { Component, Input } from '@angular/core'

import { datetimeToLocal } from '../utils'

/**
 * Component that presents subnet as a bar with a sub-bar that shows utilizations in %.
 * It also shows details in a tooltip.
 */
@Component({
    selector: 'app-subnet-bar',
    templateUrl: './subnet-bar.component.html',
    styleUrls: ['./subnet-bar.component.sass'],
})
export class SubnetBarComponent {
    _subnet: any
    style: any
    tooltip = ''

    constructor() {}

    @Input()
    set subnet(subnet) {
        this._subnet = subnet

        const util = subnet.addrUtilization ? subnet.addrUtilization : 0
        const util2 = Math.floor(util)

        const style = {
            width: util + '%',
        }

        if (util > 90) {
            style['background-color'] = '#faa' // red-ish
        } else if (util > 80) {
            style['background-color'] = '#ffcf76' // orange-ish
        } else {
            style['background-color'] = '#abffa8' // green-ish
        }

        this.style = style

        if (this._subnet.localSubnets[0].stats) {
            const stats = this._subnet.localSubnets[0].stats
            const lines = []
            if (this._subnet.subnet.includes('.')) {
                // DHCPv4 stats
                lines.push('Total: ' + stats['total-addreses'].toLocaleString('en-US'))
                lines.push('Assigned: ' + stats['assigned-addreses'].toLocaleString('en-US'))
                lines.push('Declined: ' + stats['declined-addreses'].toLocaleString('en-US'))
            } else {
                // DHCPv6 stats
                // NAs
                if (stats['total-nas'] !== undefined) {
                    let total = stats['total-nas']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total NAs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-nas'] !== undefined) {
                    lines.push('Assigned NAs: ' + stats['assigned-nas'].toLocaleString('en-US'))
                }
                if (stats['declined-nas'] !== undefined) {
                    lines.push('Declined NAs: ' + stats['declined-nas'].toLocaleString('en-US'))
                }
                // PDs
                if (stats['total-pds'] !== undefined) {
                    let total = stats['total-pds']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total PDs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-pds'] !== undefined) {
                    lines.push('Assigned PDs: ' + stats['assigned-pds'].toLocaleString('en-US'))
                }
            }
            lines.push('Collected at: ' + datetimeToLocal(this._subnet.localSubnets[0].statsCollectedAt))
            this.tooltip = lines.join('<br>')
        } else {
            this.tooltip = 'No stats yet'
            style['background-color'] = '#ccc' // grey
        }
    }

    get subnet() {
        return this._subnet
    }
}
