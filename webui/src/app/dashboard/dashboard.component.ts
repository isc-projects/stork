import { Component, OnInit } from '@angular/core'

import { MessageService } from 'primeng/api'

import { ServicesService, DHCPService } from '../backend/api/api'
import { AppsStats } from '../backend/model/appsStats'
import { humanCount, durationToString } from '../utils'
import { SettingService } from '../setting.service'

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
        private servicesApi: ServicesService,
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
        this.servicesApi.getAppsStats().subscribe(
            data => {
                this.loaded = true
                this.appsStats = { ...this.appsStats, ...data }
            },
            err => {
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

        this.settingSvc.getSettings().subscribe(data => {
            this.grafanaUrl = data['grafana_url']
        })
    }

    /**
     * Get or refresh DHCP overview data from the server
     */
    refreshDhcpOverview() {
        this.dhcpApi.getDhcpOverview().subscribe(
            data => {
                this.overview = data
            },
            err => {
                this.loaded = true
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
}
