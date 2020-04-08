import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'

import { MessageService } from 'primeng/api'

import { ServicesService, DHCPService } from '../backend/api/api'
import { AppsStats } from '../backend/model/appsStats'
import { humanCount, durationToString } from '../utils'

@Component({
    selector: 'app-dashboard',
    templateUrl: './dashboard.component.html',
    styleUrls: ['./dashboard.component.sass'],
})
export class DashboardComponent implements OnInit {
    loaded = false
    appsStats: AppsStats
    overview: any

    constructor(
        private router: Router,
        private servicesApi: ServicesService,
        private dhcpApi: DHCPService,
        private msgSrv: MessageService
    ) {}

    ngOnInit() {
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

        this.dhcpApi.getDhcpOverview().subscribe(
            data => {
                console.info(data)
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
                    summary: 'Cannot get applications statistics',
                    detail: 'Getting applications statistics erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

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

    humanCount(num) {
        return humanCount(num)
    }

    showDuration(duration) {
        return durationToString(duration)
    }
}
