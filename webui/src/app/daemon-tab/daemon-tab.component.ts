import { Component, EventEmitter, Input, Output } from '@angular/core'

import { AnyDaemon } from '../backend'
import { daemonStatusIconColor, daemonStatusIconName, daemonStatusIconTooltip } from '../utils'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { KeaDaemonComponent } from '../kea-daemon/kea-daemon.component'
import { Bind9DaemonComponent } from '../bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from '../pdns-daemon/pdns-daemon.component'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { Button } from 'primeng/button'
import { Tooltip } from 'primeng/tooltip'
import { Panel } from 'primeng/panel'
import { RouterLink } from '@angular/router'
import { isKeaDaemon } from '../version.service'

@Component({
    selector: 'app-daemon-tab',
    templateUrl: './daemon-tab.component.html',
    styleUrl: './daemon-tab.component.sass',
    imports: [
        Panel,
        Tooltip,
        Button,
        DaemonNiceNamePipe,
        VersionStatusComponent,
        KeaDaemonComponent,
        Bind9DaemonComponent,
        PdnsDaemonComponent,
        EventsPanelComponent,
        PlaceholderPipe,
        LocaltimePipe,
        DurationPipe,
        RouterLink,
    ],
})
export class DaemonTabComponent {
    @Input() daemon: AnyDaemon
    @Output() refreshDaemon = new EventEmitter<number>()

    get daemonStatusIconName() {
        return daemonStatusIconName(this.daemon)
    }

    get daemonStatusIconColor() {
        return daemonStatusIconColor(this.daemon)
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
