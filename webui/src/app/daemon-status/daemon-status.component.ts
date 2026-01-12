import { Component, Input } from '@angular/core'

import { daemonStatusIconClass, daemonStatusIconTooltip } from '../utils'
import { AnyDaemon } from '../backend'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { RouterLink } from '@angular/router'
import { NgStyle } from '@angular/common'
import { Tooltip } from 'primeng/tooltip'

@Component({
    selector: 'app-daemon-status',
    templateUrl: './daemon-status.component.html',
    styleUrls: ['./daemon-status.component.sass'],
    imports: [RouterLink, DaemonNiceNamePipe, NgStyle, Tooltip],
})
export class DaemonStatusComponent {
    @Input({ required: true }) daemon: AnyDaemon

    get daemonStatusIconTooltip() {
        return daemonStatusIconTooltip(this.daemon)
    }

    get daemonStatusIconClass() {
        return daemonStatusIconClass(this.daemon)
    }
}
