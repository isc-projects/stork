import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { MessageService } from 'primeng/api'
import { ServicesService } from '../backend/api/api'

/**
 * The component comprising a list of configuration review
 * reports for a daemon.
 *
 * The Stork server reviews the configurations of the monitored
 * servers using built-in checkers. Each checker verifies some
 * aspect or part of the configuration. It tries to find
 * configuration errors or suggestions for configuration changes
 * to improve Stork's monitoring capabilities of that server.
 * The component fetches the review reports for a specified
 * daemon from the server and displays them. Each report comes
 * with a checker name displayed in the blue badge. The checker
 * names are provided to make it easier to distinguish between
 * different issues without reading sometimes lengthy reports.
 * The displayed list has pagination capabilities.
 */
@Component({
    selector: 'app-config-review-panel',
    templateUrl: './config-review-panel.component.html',
    styleUrls: ['./config-review-panel.component.sass'],
})
export class ConfigReviewPanelComponent implements OnInit {
    /**
     * ID of a daemon for which reports are listed.
     */
    @Input() daemonId: number

    /**
     * Event emitter notifying a parent component that the total number
     * of reports has been updated.
     *
     * The event comprises daemon id and the total reports number.
     */
    @Output() updateTotal = new EventEmitter<{ daemonId: number; total: number }>()

    /**
     * List pagination offset.
     */
    start = 0

    /**
     * A number of reports per page.
     */
    limit = 5

    /**
     * Total number of reports available for a daemon.
     */
    total = 0

    /**
     * The currently displayed reports.
     */
    reports: any[] = []

    /**
     * Component constructor.
     *
     * @param msgService a service used to display error messages.
     * @param servicesApi a service used to fetch the config review reports.
     */
    constructor(private msgService: MessageService, private servicesApi: ServicesService) {}

    /**
     * A hook invoked during the component initialization.
     *
     * It fetches the list of the configuration reports from the first
     * up to the limit per page.
     */
    ngOnInit(): void {
        this.refreshDaemonConfigReports(null)
    }

    /**
     * Fetches the configuration reports using pagination.
     *
     * @param event an event emitted when user navigates over the pages;
     * it comprises the offset and limit of reports to fetch.
     */
    refreshDaemonConfigReports(event) {
        if (event) {
            this.start = event.first
            this.limit = event.rows
        } else {
            this.start = 0
            this.limit = 5
        }
        // Get reports with specifying the limits.
        this.servicesApi
            .getDaemonConfigReports(this.daemonId, this.start, this.limit)
            .toPromise()
            .then((data) => {
                this.reports = data.items
                this.setTotal(data.total)
            })
            .catch((err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgService.add({
                    severity: 'error',
                    summary: 'Getting review reports erred',
                    detail: 'Getting review reports erred: ' + msg,
                    life: 10000,
                })
                this.start = 0
                this.limit = 5
                this.setTotal(0)
                this.reports = []
            })
    }

    /**
     * Event handling function invoked when the user navigates over the
     * review report pages.
     *
     * It fetches a page of review reports.
     *
     * @param event an event emitted when user navigates over the pages;
     * it comprises the offset and limit of reports to fetch.
     */
    paginate(event) {
        this.refreshDaemonConfigReports(event)
    }

    /**
     * Sets new total reports number and emits an event.
     *
     * @param total new total reports number.
     */
    setTotal(total: number) {
        this.total = total
        this.updateTotal.emit({ daemonId: this.daemonId, total: this.total })
    }
}
