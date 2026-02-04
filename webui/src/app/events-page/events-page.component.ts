import { Component, OnInit } from '@angular/core'
import { ActivatedRoute } from '@angular/router'
import { Daemon } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { EventsPanelComponent } from '../events-panel/events-panel.component'

/**
 * Component responsible for showing a full page list of events,
 * with filtering and pagination capabilities.
 */
@Component({
    selector: 'app-events-page',
    templateUrl: './events-page.component.html',
    styleUrls: ['./events-page.component.sass'],
    imports: [BreadcrumbsComponent, EventsPanelComponent],
})
export class EventsPageComponent implements OnInit {
    machineId: number = null
    daemonName: Daemon.NameEnum = null
    userId: number = null
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Events' }]

    constructor(private route: ActivatedRoute) {}

    ngOnInit(): void {
        const machineId = this.route.snapshot.queryParams.machineId
        if (machineId) {
            this.machineId = parseInt(machineId, 10)
        }

        const daemonName = this.route.snapshot.queryParams.daemonName
        if (daemonName) {
            this.daemonName = daemonName as Daemon.NameEnum
        }

        const userId = this.route.snapshot.queryParams.userId
        if (userId) {
            this.userId = parseInt(userId, 10)
        }
    }
}
