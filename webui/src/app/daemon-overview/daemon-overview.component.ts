import { Component, Input } from '@angular/core'
import { AnyDaemon } from '../backend'
import { NgIf, NgFor } from '@angular/common'
import { RouterLink } from '@angular/router'
import { Tooltip } from 'primeng/tooltip'
import { ManagedAccessDirective } from '../managed-access.directive'
import { AccessPointKeyComponent } from '../access-point-key/access-point-key.component'

/**
 * A component that displays daemon overview.
 *
 * It comprises the information about the daemon and machine access points.
 */
@Component({
    selector: 'app-daemon-overview',
    templateUrl: './daemon-overview.component.html',
    styleUrls: ['./daemon-overview.component.sass'],
    imports: [NgIf, RouterLink, NgFor, Tooltip, ManagedAccessDirective, AccessPointKeyComponent],
})
export class DaemonOverviewComponent {
    /** Pointer to the structure holding the daemon information. */
    @Input() daemon: AnyDaemon

    /**
     * Conditionally formats an IP address for display.
     *
     * @param addr IPv4 or IPv6 address string.
     * @returns unchanged value if it is an IPv4 address or an IPv6 address
     *          surrounded by [ ].
     */
    formatAddress(addr: string): string {
        if (addr.length === 0 || !addr.includes(':') || (addr.startsWith('[') && addr.endsWith(']')))
            return addr

        return `[${addr}]`
    }
}
