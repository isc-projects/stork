import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'
import { ActivatedRoute } from '@angular/router'
import { BehaviorSubject } from 'rxjs'

import moment from 'moment-timezone'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'

import {
    durationToString,
    daemonStatusErred,
    daemonStatusIconName,
    daemonStatusIconColor,
    daemonStatusIconTooltip,
} from '../utils'

@Component({
    selector: 'app-kea-app-tab',
    templateUrl: './kea-app-tab.component.html',
    styleUrls: ['./kea-app-tab.component.sass'],
})
export class KeaAppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()
    @Input() refreshedAppTab: any

    daemons: any[] = []

    activeTabIndex = 0

    constructor(private route: ActivatedRoute, private servicesApi: ServicesService) {}

    /**
     * Subscribes to the updates of the information about daemons
     *
     * The information about the daemons may be updated as a result of
     * pressing the refresh button in the app tab. In such case, this
     * component emits an event to which the parent component reacts
     * and updates the daemons. When the daemons are updated, it
     * notifies this compoment via the subscription mechanism.
     */
    ngOnInit() {
        this.refreshedAppTab.subscribe((data) => {
            if (data) {
                this.initDaemons(data.app.details.daemons)
            }
        })
    }

    /**
     * Selects new application tab
     *
     * As a result, the local information about the daemons is updated.
     *
     * @param appTab pointer to the new app tab data structure.
     */
    @Input()
    set appTab(appTab) {
        this._appTab = appTab
        // Refresh local information about the daemons presented by this
        // component.
        this.initDaemons(appTab.app.details.daemons)
    }

    /**
     * Returns information about currently selected app tab.
     */
    get appTab() {
        return this._appTab
    }

    /**
     * Initializes information about the daemons according to the information
     * carried in the provided parameter.
     *
     * As a result of invoking this function, the view of the component will be
     * updated.
     *
     * @param appTabDaemons information about the daemons stored in the app tab
     *                      data structure.
     */
    private initDaemons(appTabDaemons) {
        const activeDaemonTabName = this.route.snapshot.queryParams.daemon || null
        const daemonMap = []
        for (const d of appTabDaemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [
            ['dhcp4', 'DHCPv4'],
            ['dhcp6', 'DHCPv6'],
            ['d2', 'DDNS'],
            ['ca', 'CA'],
            ['netconf', 'NETCONF'],
        ]
        const daemons = []
        let idx = 0
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemonMap[dm[0]].subnets = []
                daemonMap[dm[0]].totalSubnets = 0
                daemonMap[dm[0]].statusErred = this.daemonStatusErred(daemonMap[dm[0]])
                daemons.push(daemonMap[dm[0]])

                if (dm[0] === activeDaemonTabName) {
                    this.activeTabIndex = idx
                }
                idx += 1
            }
        }
        this.daemons = daemons
    }

    /**
     * An action triggered when refresh button is pressed.
     */
    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
    }

    /**
     * Converts duration to pretty string.
     *
     * @param duration duration value to be converted.
     *
     * @returns duration as text
     */
    showDuration(duration) {
        return durationToString(duration)
    }

    /**
     * Returns boolean value indicating if there is an issue with communication
     * with the active daemon.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @return true if there is a communication problem with the daemon,
     *         false otherwise.
     */
    private daemonStatusErred(daemon): boolean {
        if (daemon.active && daemonStatusErred(daemon)) {
            return true
        }
        return false
    }

    /**
     * Returns the name of the icon to be used when presenting daemon status
     *
     * The icon selected depends on whether the daemon is active or not
     * active and whether there is a communication with the daemon or
     * not.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns ban icon if the daemon is not active, times icon if the daemon
     *          should be active but the communication with it is borken and
     *          check icon if the communication with the active daemon is ok.
     */
    daemonStatusIconName(daemon) {
        return daemonStatusIconName(daemon)
    }

    /**
     * Returns the color of the icon used when presenting daemon status
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
     * Returns error text to be displayed when there is a communication issue
     * with a given daemon
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns Error text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    daemonStatusErrorText(daemon) {
        return daemonStatusIconTooltip(daemon)
    }

    changeMonitored(daemon) {
        const dmn = { monitored: !daemon.monitored }
        this.servicesApi.updateDaemon(daemon.id, dmn).subscribe(
            (data) => {
                daemon.monitored = dmn.monitored
            },
            (err) => {
                console.warn('failed to update monitoring flag in daemon')
            }
        )
    }

    /**
     * Checks if the specified log target can be viewed
     *
     * Only the logs that are stored in the file can be viewed in Stork. The
     * logs output to stdout, stderr or syslog can't be viewed in Stork.
     *
     * @param target log target output location
     * @returns true if the log target can be viewed, false otherwise.
     */
    logTargetViewable(target): boolean {
        return target !== 'stdout' && target !== 'stderr' && !target.startsWith('syslog')
    }
}
