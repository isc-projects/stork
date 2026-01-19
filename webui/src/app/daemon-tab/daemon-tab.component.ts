import { Component, EventEmitter, Input, Output } from '@angular/core'

import { AnyDaemon } from '../backend'
import { daemonStatusIconClass, daemonStatusIconTooltip } from '../utils'
import { KeaDaemonComponent } from '../kea-daemon/kea-daemon.component'
import { Bind9DaemonComponent } from '../bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from '../pdns-daemon/pdns-daemon.component'
import { Button } from 'primeng/button'
import { Tooltip } from 'primeng/tooltip'
import { isKeaDaemon } from '../version.service'
import { EntityLinkComponent } from '../entity-link/entity-link.component'

@Component({
    selector: 'app-daemon-tab',
    templateUrl: './daemon-tab.component.html',
    styleUrl: './daemon-tab.component.sass',
    imports: [Tooltip, Button, KeaDaemonComponent, Bind9DaemonComponent, PdnsDaemonComponent, EntityLinkComponent],
})
export class DaemonTabComponent {
    @Input() daemon: AnyDaemon
    @Output() refreshDaemon = new EventEmitter<number>()

    /**
     * Returns the CSS class to display the icon to be used to indicate daemon status
     */
    get daemonStatusIconClass() {
        return daemonStatusIconClass(this.daemon)
    }

    /**
     * Returns tooltip for the icon presented for the daemon status
     */
    get daemonStatusIconTooltip() {
        return daemonStatusIconTooltip(this.daemon)
    }

    /**
     * Indicates if the given daemon is a Kea daemon.
     * @param daemon
     * @returns true if the daemon is Kea daemon; otherwise false.
     */
    get isKeaDaemon() {
        return isKeaDaemon(this.daemon?.name)
    }

    /**
     * Emits the refresh event.
     */
    refresh() {
        if (this.daemon?.id !== undefined) {
            this.refreshDaemon.emit(this.daemon.id)
        }
    }
}
