import { Component, Input } from '@angular/core'
import { App } from '../backend'

/**
 * A component that displays app overview.
 *
 * It comprises the information about the app and machine access points.
 */
@Component({
    selector: 'app-app-overview',
    standalone: false,
    templateUrl: './app-overview.component.html',
    styleUrls: ['./app-overview.component.sass'],
})
export class AppOverviewComponent {
    /**
     * Pointer to the structure holding the app information.
     */
    @Input() app: App = null

    /**
     * Conditionally formats an IP address for display.
     *
     * @param addr IPv4 or IPv6 address string.
     * @returns unchanged value if it is an IPv4 address or an IPv6 address
     *          surrounded by [ ].
     */
    formatAddress(addr: string): string {
        if (addr.length === 0 || !addr.includes(':') || (addr.startsWith('[') && addr.endsWith(']'))) {
            return addr
        }
        return `[${addr}]`
    }
}
