import { Component, Input } from '@angular/core'

import { daemonStatusIconName, daemonStatusIconColor, daemonStatusIconTooltip } from '../utils'
import { App } from '../backend'

@Component({
    selector: 'app-app-daemons-status',
    templateUrl: './app-daemons-status.component.html',
    styleUrls: ['./app-daemons-status.component.sass'],
})
export class AppDaemonsStatusComponent {
    @Input() app: any

    constructor() {}

    /** Returns a list of daemon sorted using custom rules. */
    sortDaemonsByImportance(app: App) {
        const daemonMap = []
        const daemons = []

        if (app.details.daemons) {
            for (const d of app.details.daemons) {
                daemonMap[d.name] = d
            }
            const DMAP = [
                ['dhcp4', 'DHCPv4'],
                ['dhcp6', 'DHCPv6'],
                ['d2', 'DDNS'],
                ['ca', 'CA'],
                ['netconf', 'NETCONF'],
            ]
            for (const dm of DMAP) {
                if (daemonMap[dm[0]] !== undefined) {
                    daemonMap[dm[0]].niceName = dm[1]
                    daemons.push(daemonMap[dm[0]])
                }
            }
        } else if (app.details.daemon) {
            daemonMap[app.details.daemon.name] = app.details.daemon
            const DMAP = [['named', 'named']]
            for (const dm of DMAP) {
                if (daemonMap[dm[0]] !== undefined) {
                    daemonMap[dm[0]].niceName = dm[1]
                    daemons.push(daemonMap[dm[0]])
                }
            }
        }

        return daemons
    }

    /**
     * Returns tooltip for the icon in presenting daemon status
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns Tooltip as text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    daemonStatusIconTooltip(daemon) {
        return daemonStatusIconTooltip(daemon)
    }

    /**
     * Returns the color of the icon used in presenting daemon status
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns grey color if the daemon is not active, red if the daemon is
     *          active but there are communication issues, green if the
     *          communication with the active daemon is ok.
     */
    daemonStatusIconColor(daemon) {
        return daemonStatusIconColor(daemon)
    }

    /**
     * Returns the name of the icon used in presenting daemon status
     *
     * The icon selected depends on whether the daemon is active or not
     * active and whether there is a communication with the daemon or
     * not.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns ban icon if the daemon is not active, times icon if the daemon
     *          should be active but the communication with it is broken and
     *          check icon if the communication with the active daemon is ok.
     */
    daemonStatusIconName(daemon) {
        return daemonStatusIconName(daemon)
    }
}
