import { Component, computed, input } from '@angular/core'

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
    daemon = input<AnyDaemon>(null)

    /**
     * Tooltip for the icon presented for the daemon status
     */
    daemonStatusIconTooltip = computed(() => daemonStatusIconTooltip(this.daemon()))

    /**
     * The CSS class to display the icon to be used to indicate daemon status
     */
    daemonStatusIconClass = computed(() => daemonStatusIconClass(this.daemon()))
}
