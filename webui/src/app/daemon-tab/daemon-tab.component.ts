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

    isKeaDaemon(daemon: AnyDaemon) {
        const keaDaemons = ['dhcp4', 'dhcp6', 'd2', 'ca', 'netconf']
        return keaDaemons.includes(daemon?.name)
    }

    appTypeForEvents(daemon: AnyDaemon) {
        if (this.isKeaDaemon(daemon)) {
            return 'kea'
        }

        if (daemon?.name === 'bind9') {
            return 'bind9'
        }

        return null
    }

    refresh() {
        if (this.daemon?.id !== undefined) {
            this.refreshDaemon.emit(this.daemon.id)
        }
    }
}
