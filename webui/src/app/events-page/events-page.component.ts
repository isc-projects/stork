import { Component, OnInit } from '@angular/core'
import { ActivatedRoute } from '@angular/router'

/**
 * Component responsible for showing a full page list of events,
 * with filtering and pagination capabilities.
 */
@Component({
    selector: 'app-events-page',
    templateUrl: './events-page.component.html',
    styleUrls: ['./events-page.component.sass'],
})
export class EventsPageComponent implements OnInit {
    machineId = null
    appType = null
    daemonType = null
    userId = null
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Events' }]

    constructor(private route: ActivatedRoute) {}

    ngOnInit(): void {
        const machineId = this.route.snapshot.queryParams.machine
        if (machineId) {
            this.machineId = parseInt(machineId, 10)
        }

        const appType = this.route.snapshot.queryParams.appType
        if (appType) {
            this.appType = appType
        }

        const daemonType = this.route.snapshot.queryParams.daemonType
        if (daemonType) {
            this.daemonType = daemonType
        }

        const userId = this.route.snapshot.queryParams.user
        if (userId) {
            this.userId = parseInt(userId, 10)
        }
    }
}
