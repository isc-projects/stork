import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'

import { MessageService } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { AppsStats } from '../backend/model/appsStats'

@Component({
    selector: 'app-dashboard',
    templateUrl: './dashboard.component.html',
    styleUrls: ['./dashboard.component.sass'],
})
export class DashboardComponent implements OnInit {
    appsStats: AppsStats

    constructor(private router: Router, private servicesApi: ServicesService, private msgSrv: MessageService) {}

    ngOnInit() {
        this.appsStats = {
            keaAppsTotal: 0,
            keaAppsNotOk: 0,
        }

        this.servicesApi.getAppsStats().subscribe(
            data => {
                this.appsStats = { ...this.appsStats, ...data }
                if (this.appsStats.keaAppsTotal === 0) {
                    // redirect to machines page so user can add some machine
                    this.router.navigate(['/machines/all'])
                }
            },
            err => {
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
}
