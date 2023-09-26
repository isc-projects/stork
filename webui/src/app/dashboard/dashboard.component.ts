import { Component, OnDestroy, OnInit } from '@angular/core'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend/api/api'
import { AppsStats } from '../backend/model/appsStats'
import {
    datetimeToLocal,
    durationToString,
    getGrafanaUrl,
    daemonStatusIconName,
    daemonStatusIconColor,
    daemonStatusIconTooltip,
    getGrafanaSubnetTooltip,
    getErrorMessage,
} from '../utils'
import { SettingService } from '../setting.service'
import { ServerDataService } from '../server-data.service'
import { Subscription } from 'rxjs'
import { map } from 'rxjs/operators'
import { parseSubnetsStatisticValues } from '../subnets'
import { DhcpDaemon, DhcpOverview, LocalSubnet, Subnet, Subnets } from '../backend'
import { ModifyDeep } from '../utiltypes'

type DhcpOverviewParsed = ModifyDeep<
    DhcpOverview,
    {
        subnets4: {
            items: {
                localSubnets: {
                    // The UI presents whole subnet or shared network statistics.
                    // They shouldn't be recalculated on the frontend because
                    // it's not trivial, and we don't want to double the
                    // calculation logic on the front and backend.
                    // The local subnets are marked as never
                    // to prevent recalculation. It causes any usage of local
                    // subnets to be labeled as an error. If we don't find any
                    // use case for them, then it will be easier to stop
                    // sharing them between frontend and backend.
                    stats: never
                    statsCollectedAt: never
                }[]
                stats: Record<string, bigint | number>
            }[]
        }
        subnets6: {
            items: {
                localSubnets: {
                    // Prevents local subnets recalculation.
                    stats: never
                    statsCollectedAt: never
                }[]
                stats: Record<string, bigint | number>
            }[]
        }
        dhcp4Stats: {
            assignedAddresses: bigint | number
            totalAddresses: bigint | number
            declinedAddresses: bigint | number
        }
        dhcp6Stats: {
            assignedNAs: bigint | number
            totalNAs: bigint | number
            assignedPDs: bigint | number
            totalPDs: bigint | number
        }
    }
>

/**
 * Component presenting dashboard with DHCP and DNS overview.
 */
@Component({
    selector: 'app-dashboard',
    templateUrl: './dashboard.component.html',
    styleUrls: ['./dashboard.component.sass'],
})
export class DashboardComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    loaded = false
    appsStats: AppsStats
    overview: DhcpOverviewParsed
    grafanaUrl: string

    constructor(
        private serverData: ServerDataService,
        private dhcpApi: DHCPService,
        private msgSrv: MessageService,
        private settingSvc: SettingService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    ngOnInit() {
        // prepare initial data so it can be used in html templates
        // before the actual data arrives from the server
        this.overview = {
            subnets4: { total: 0, items: [] },
            subnets6: { total: 0, items: [] },
            sharedNetworks4: { total: 0, items: [] },
            sharedNetworks6: { total: 0, items: [] },
            dhcp4Stats: { assignedAddresses: 0, totalAddresses: 0, declinedAddresses: 0 },
            dhcp6Stats: { assignedNAs: 0, totalNAs: 0, assignedPDs: 0, totalPDs: 0 },
            dhcpDaemons: [],
        }
        this.appsStats = {
            keaAppsTotal: 0,
            keaAppsNotOk: 0,
            bind9AppsTotal: 0,
            bind9AppsNotOk: 0,
        }

        // get stats about apps
        this.subscriptions.add(
            this.serverData.getAppsStats().subscribe(
                (data) => {
                    this.loaded = true
                    this.appsStats = { ...this.appsStats, ...data }
                },
                (err) => {
                    this.loaded = true
                    let msg = getErrorMessage(err)
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Cannot get applications statistics',
                        detail: 'Error getting applications statistics: ' + msg,
                        life: 10000,
                    })
                }
            )
        )

        // get DHCP overview from the server
        this.refreshDhcpOverview()

        this.subscriptions.add(
            this.settingSvc.getSettings().subscribe((data) => {
                this.grafanaUrl = data['grafana_url']
            })
        )
    }

    /**
     * Get or refresh DHCP overview data from the server
     */
    refreshDhcpOverview() {
        return this.dhcpApi
            .getDhcpOverview()
            .pipe(
                map((v) => {
                    // Convert strings to bigints in statistics
                    // DHCPv4 global statistics
                    if (v.dhcp4Stats) {
                        for (const stat of Object.keys(v.dhcp4Stats)) {
                            try {
                                v.dhcp4Stats[stat] = BigInt(v.dhcp4Stats[stat])
                            } catch {
                                continue
                            }
                        }
                    }

                    // DHCPv6 global statistics
                    if (v.dhcp6Stats) {
                        for (const stat of Object.keys(v.dhcp6Stats)) {
                            try {
                                v.dhcp6Stats[stat] = BigInt(v.dhcp6Stats[stat])
                            } catch {
                                continue
                            }
                        }
                    }

                    // IPv4 subnets statistics
                    if (v.subnets4 && v.subnets4.items) {
                        parseSubnetsStatisticValues(v.subnets4.items)
                    }

                    // IPv6 subnets statistics
                    if (v.subnets6 && v.subnets6.items) {
                        parseSubnetsStatisticValues(v.subnets6.items)
                    }

                    return v as unknown as DhcpOverviewParsed
                })
            )
            .toPromise()
            .then((data) => {
                this.overview = data
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get DHCP overview',
                    detail: 'Error getting DHCP overview: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Estimate percent based on numerator and denominator.
     */
    getPercent(numerator: number | bigint | null, denominator: number | bigint | null) {
        // Denominator or numerator is undefined, null or zero.
        if (!denominator || !numerator) {
            return 0
        }

        if (typeof numerator === 'bigint' || typeof denominator === 'bigint') {
            return Number((BigInt(100) * BigInt(numerator)) / BigInt(denominator))
        }

        const percent = (100 * numerator) / denominator
        return Math.floor(percent)
    }

    /**
     * Make duration human readable.
     */
    showDuration(duration) {
        return durationToString(duration, true)
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }

    /**
     * Build a tooltip explaining what the subnet link is for
     * @param subnet id of the subnet
     * @param machine id of the machine
     */
    getGrafanaTooltip(subnet, machine) {
        return getGrafanaSubnetTooltip(subnet, machine)
    }

    /**
     * Returns the name of the icon to be used in the Status column
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

    /**
     * Returns the color of the icon used in the Status column
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
     * Returns tooltip for the icon presented in the Status column
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
     * Returns tooltip for an RPS column
     *
     * @param daemon data structure holding the information about the daemon.
     * @param interval indicates whether this is RPS for interval 1 or 2
     *
     * @returns Tooltip as text.
     */
    daemonRpsTooltip(daemon, interval) {
        const typeStr = daemon.name === 'dhcp4' ? 'ACKs' : 'REPLYs'
        const intervalStr = interval === 1 ? '15 minutes' : '24 hours'
        return 'Number of ' + typeStr + ' sent by the daemon per second over the last ' + intervalStr
    }

    /**
     * Returns the name of the icon to be shown for the given HA state
     *
     * @param daemon daemon for which HA state is being displayed.
     * @returns check, times, exclamation triangle or ban or spinner.
     */
    haStateIcon(daemon) {
        if (!daemon.haEnabled) {
            return 'ban'
        }
        if (!daemon.haState || daemon.haState.length === 0) {
            return 'spin pi-spinner'
        }
        switch (daemon.haState) {
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
     * @param daemon daemon which state should be returned.
     * @returns state name or 'not configured' if the state name
     *          is empty or 'fetching' if the state is to be fetched.
     */
    showHAState(daemon) {
        if (!daemon.haEnabled) {
            return 'not configured'
        }
        if (!daemon.haState || daemon.haState.length === 0) {
            return 'fetching...'
        }
        return daemon.haState
    }

    /**
     * Returns printable time when failover was last triggered for a
     * given daemon.
     *
     * @param daemon daemon which last failure time should be returned.
     *
     * @returns empty string of the state is unavailable, timestamp in local
     *          time if it is non-zero or 'never' if the specified timestamp
     *          is zero.
     */
    showHAFailureTime(daemon: DhcpDaemon) {
        if (!daemon.haEnabled || !daemon.haState) {
            return ''
        }
        const localTime = datetimeToLocal(daemon.haFailureAt)
        if (!localTime) {
            return 'never'
        }
        return localTime
    }
}
