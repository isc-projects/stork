import { Component, Input } from '@angular/core'

import { daemonStatusIconClass, daemonStatusIconTooltip } from '../utils'
import { AnyDaemon } from '../backend'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { RouterLink } from '@angular/router'
import { Tooltip } from 'primeng/tooltip'

@Component({
    selector: 'app-daemon-status',
    templateUrl: './daemon-status.component.html',
    styleUrls: ['./daemon-status.component.sass'],
    imports: [RouterLink, DaemonNiceNamePipe, Tooltip],
})
export class DaemonStatusComponent {
    @Input({ required: true }) daemon: AnyDaemon

    /**
     * Returns tooltip for the icon presented for the daemon status
     */
    get daemonStatusIconTooltip() {
        return daemonStatusIconTooltip(this.daemon)
    }

    /**
     * Returns the CSS class to display the icon to be used to indicate daemon status
     */
    get daemonStatusIconClass() {
        return daemonStatusIconClass(this.daemon)
    }
}
