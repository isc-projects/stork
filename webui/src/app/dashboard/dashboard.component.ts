import { Component, OnInit } from '@angular/core'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend/api/api'
import { AppsStats } from '../backend/model/appsStats'
import { datetimeToLocal, durationToString, getGrafanaUrl, humanCount } from '../utils'
import { SettingService } from '../setting.service'
import { ServerDataService } from '../server-data.service'

/**
 * Component presenting dashboard with DHCP and DNS overview.
 */
@Component({
    selector: 'app-dashboard',
    templateUrl: './dashboard.component.html',
    styleUrls: ['./dashboard.component.sass'],
})
export class DashboardComponent implements OnInit {
    loaded = false
    appsStats: AppsStats
    overview: any
    grafanaUrl: string

    constructor(
        private serverData: ServerDataService,
        private dhcpApi: DHCPService,
        private msgSrv: MessageService,
        private settingSvc: SettingService
    ) {}

    ngOnInit() {
        // prepare initial data so it can be used in html templates
        // before the actual data arrives from the server
        this.overview = {
            subnets4: { total: 0, items: [] },
            subnets6: { total: 0, items: [] },
            sharedNetworks4: { total: 0, items: [] },
            sharedNetworks6: { total: 0, items: [] },
            dhcp4Stats: { assignedAddresses: 0, totalAddresses: 0, declinedAddresses: 0 },
            dhcp6Stats: { assignedNAs: 0, totalNAs: 0, assignedPDs: 0, totalPDs: 0, declinedAddresses: 0 },
            dhcpDaemons: [],
        }
        this.appsStats = {
            keaAppsTotal: 0,
            keaAppsNotOk: 0,
            bind9AppsTotal: 0,
            bind9AppsNotOk: 0,
        }

        // get stats about apps
        this.serverData.getAppsStats().subscribe(
            (data) => {
                this.loaded = true
                this.appsStats = { ...this.appsStats, ...data }
            },
            (err) => {
                this.loaded = true
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get applications statistics',
                    detail: 'Getting applications statistics erred: ' + msg,
                    life: 10000,
                })
            }
        )

        // get DHCP overview from the server
        this.refreshDhcpOverview()

        this.settingSvc.getSettings().subscribe((data) => {
            this.grafanaUrl = data['grafana_url']
        })
    }

    /**
     * Get or refresh DHCP overview data from the server
     */
    refreshDhcpOverview() {
        this.dhcpApi.getDhcpOverview().subscribe(
            (data) => {
                this.overview = data
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get DHCP overview',
                    detail: 'Getting DHCP overview erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Estimate percent based on numerator and denominator.
     */
    getPercent(numerator, denominator) {
        if (denominator === undefined) {
            return 0
        }
        if (numerator === undefined) {
            numerator = 0
        }
        const percent = (100 * numerator) / denominator
        return Math.floor(percent)
    }

    /**
     * Make number human readable.
     */
    humanCount(num) {
        return humanCount(num)
    }

    /**
     * Make duration human readable.
     */
    showDuration(duration) {
        return durationToString(duration)
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }

    /**
     * Returns the name of the icon to be used in the Active column
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
    activeIconName(daemon) {
        if (!daemon.active) {
            return 'ban'
        }
        if (
            (daemon.agentCommErrors && daemon.agentCommErrors > 0) ||
            (daemon.caCommErrors && daemon.caCommErrors > 0) ||
            (daemon.daemonCommErrors && daemon.daemonCommErrors > 0)
        ) {
            return 'times'
        }
        return 'check'
    }

    /**
     * Returns the color of the icon used in the Active column
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns grey color if the daemon is not active, red if the daemon is
     *          active but there are communication issues, green if the
     *          communication with the active daemon is ok.
     */
    activeIconColor(daemon) {
        if (!daemon.active) {
            return 'grey'
        }
        if (
            (daemon.agentCommErrors && daemon.agentCommErrors > 0) ||
            (daemon.caCommErrors && daemon.caCommErrors > 0) ||
            (daemon.daemonCommErrors && daemon.daemonCommErrors > 0)
        ) {
            return '#f11'
        }
        return '#00a800'
    }

    /**
     * Returns tooltip for the icon presented in the Active column
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns Tooltip as text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    activeIconTooltip(daemon) {
        if (!daemon.active) {
            return (
                'Configuration for this daemon was found but the daemon was ' +
                'not running at the time when the application was detected.'
            )
        }
        if (daemon.agentCommErrors && daemon.agentCommErrors > 0) {
            return (
                'Communication with the Stork Agent on this machine ' +
                'is broken. Last ' +
                daemon.agentCommErrors +
                ' attempt(s) to ' +
                'communicate with the agent failed.'
            )
        }
        if (daemon.caCommErrors && daemon.caCommErrors > 0) {
            return (
                'Communication with the Kea Control Agent on this machine ' +
                'is broken. The Stork Agent appears to work fine but the ' +
                'Kea CA is down or returns errors. Last ' +
                daemon.caCommErrors +
                ' attempt(s) to communicate with Kea CA failed.'
            )
        }
        if (daemon.daemonCommErrors && daemon.daemonCommErrors > 0) {
            return (
                'Communication with the daemon on this machine ' +
                'is broken. The Stork Agent and Kea Control Agent appear to ' +
                'work fine, but the daemon behind Kea CA does not respond or ' +
                'responds with errors. Last ' +
                daemon.daemonCommErrors +
                ' attempt(s) to communicate with the daemon failed.'
            )
        }
        return 'Communication with the daemon is ok.'
    }

    /**
     * Returns the name of the icon to be shown for the given HA state
     *
     * @returns check, times, exclamation triangle or ban.
     */
    haStateIcon(haState) {
        if (!haState || haState.length === 0) {
            return 'ban'
        }
        switch (haState) {
            case 'load-balancing':
            case 'hot-standby':
            case 'backup':
                return 'check'
            case 'unavailable':
            case 'terminated':
                return 'times'
            default:
                return 'exclamation-triangle'
        }
    }

    /**
     * Returns icon color for the given icon name.
     *
     * @returns Green color for icon check, red for times, orange for
     *          exclamation triangle, grey otherwise.
     */
    haStateIconColor(haStateIcon) {
        switch (haStateIcon) {
            case 'check':
                return '#00a800'
            case 'times':
                return '#f11'
            case 'exclamation-triangle':
                return 'orange'
            default:
                return 'grey'
        }
    }

    /**
     * Returns printable HA state value.
     *
     * @returns state name or 'not configured' if the state name
     *          is empty.
     */
    showHAState(state) {
        if (!state || state.length === 0) {
            return 'not configured'
        }
        return state
    }

    /**
     * Returns printable time when failover was last triggered for a
     * given daemon.
     *
     * @param state current HA state of the daemon.
     * @param t timestamp of the last failure.
     *
     * @returns empty string of the state is unavailable, timestamp in local
     *          time if it is non-zero or 'never' if the specified timestamp
     *          is zero.
     */
    showHAFailureTime(state, t) {
        if (!state || state.length === 0) {
            return ''
        }
        const localTime = datetimeToLocal(t)
        if (localTime.length === 0) {
            return 'never'
        }
        return localTime
    }
}
