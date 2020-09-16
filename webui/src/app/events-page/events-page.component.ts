import { Component, OnInit } from '@angular/core'
import { ActivatedRoute } from '@angular/router'

@Component({
    selector: 'app-events-page',
    templateUrl: './events-page.component.html',
    styleUrls: ['./events-page.component.sass'],
})
export class EventsPageComponent implements OnInit {
    machineId = null
    appId = null
    daemonId = null
    userId = null

    constructor(private route: ActivatedRoute) {}

    ngOnInit(): void {
        const machineId = this.route.snapshot.queryParams.machine
        if (machineId) {
            this.machineId = parseInt(machineId, 10)
        }

        const appId = this.route.snapshot.queryParams.app
        if (appId) {
            this.appId = parseInt(appId, 10)
        }

        const daemonId = this.route.snapshot.queryParams.daemon
        if (daemonId) {
            this.daemonId = parseInt(daemonId, 10)
        }

        const userId = this.route.snapshot.queryParams.user
        if (userId) {
            this.userId = parseInt(userId, 10)
        }
    }
}
