import { Component, Input } from '@angular/core'

import { getSubnetUtilization, datetimeToLocal } from '../utils'

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

        const util = getSubnetUtilization(this._subnet)

        const style = {
            width: util + '%',
        }

        if (util > 90) {
            style['background-color'] = '#faa' // red-ish
        } else if (util > 80) {
            style['background-color'] = '#ffcf76' // orange-ish
        }

        this.style = style

        if (this._subnet.stats) {
            const stats = this._subnet.stats
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
                    let total = this._subnet.stats['total-nas']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total NAs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-nas'] !== undefined) {
                    lines.push('Assigned NAs: ' + this._subnet.stats['assigned-nas'].toLocaleString('en-US'))
                }
                if (stats['declined-nas'] !== undefined) {
                    lines.push('Assigned NAs: ' + this._subnet.stats['declined-nas'].toLocaleString('en-US'))
                }
                // PDs
                if (stats['total-pds'] !== undefined) {
                    let total = this._subnet.stats['total-pds']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total PDs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-pds'] !== undefined) {
                    lines.push('Assigned PDs: ' + this._subnet.stats['assigned-pds'].toLocaleString('en-US'))
                }
            }
            lines.push('Collected at: ' + datetimeToLocal(this._subnet.statsCollectedAt))
            this.tooltip = lines.join('<br>')
        } else {
            this.tooltip = 'No stats yet'
        }
    }

    get subnet() {
        return this._subnet
    }
}
