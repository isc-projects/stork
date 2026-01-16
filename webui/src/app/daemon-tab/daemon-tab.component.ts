import { Component, EventEmitter, Input, Output } from '@angular/core'

import { AnyDaemon } from '../backend'
import { daemonStatusIconClass, daemonStatusIconTooltip } from '../utils'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { KeaDaemonComponent } from '../kea-daemon/kea-daemon.component'
import { Bind9DaemonComponent } from '../bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from '../pdns-daemon/pdns-daemon.component'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { Button } from 'primeng/button'
import { Tooltip } from 'primeng/tooltip'
import { Panel } from 'primeng/panel'
import { RouterLink } from '@angular/router'
import { isKeaDaemon } from '../version.service'
import { DaemonOverviewComponent } from '../daemon-overview/daemon-overview.component'

@Component({
    selector: 'app-daemon-tab',
    templateUrl: './daemon-tab.component.html',
    styleUrl: './daemon-tab.component.sass',
    imports: [
        Panel,
        Tooltip,
        Button,
        DaemonNiceNamePipe,
        KeaDaemonComponent,
        Bind9DaemonComponent,
        PdnsDaemonComponent,
        EventsPanelComponent,
        RouterLink,
        DaemonOverviewComponent,
    ],
})
export class DaemonTabComponent {
    @Input() daemon: AnyDaemon
    @Output() refreshDaemon = new EventEmitter<number>()

    get daemonStatusIconClass() {
        return daemonStatusIconClass(this.daemon)
    }

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
