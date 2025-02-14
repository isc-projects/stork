import { HttpStatusCode } from '@angular/common/http'
import { Component, Input, OnInit } from '@angular/core'
import { MessageService } from 'primeng/api'
import { ServicesService } from '../backend/api/api'
import { of } from 'rxjs'
import { concatMap, delay, map, retryWhen, take } from 'rxjs/operators'
import { getErrorMessage } from '../utils'
import { ConfigReport, ConfigReview } from '../backend'

/**
 * The component comprises a list of configuration review
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
 *
 * The component also allows for manually triggering the review.
 * In this case, it waits for the review to complete and then
 * refreshes the displayed configuration reports.
 */
@Component({
    selector: 'app-config-review-panel',
    templateUrl: './config-review-panel.component.html',
    styleUrls: ['./config-review-panel.component.sass'],
})
export class ConfigReviewPanelComponent implements OnInit {
    /**
     * ID of the daemon for which reports are listed.
     */
    @Input() daemonId: number

    /**
     * List pagination offset.
     */
    start = 0

    /**
     * The number of reports per page.
     */
    limit = 5

    /**
     * Total number of reports available for the current criteria.
     */
    total = 0

    /**
     * The currently displayed reports.
     */
    reports: ConfigReport[] = []

    /**
     * The currently displayed review summary.
     *
     * It contains the review generation time.
     */
    review: ConfigReview = null

    /**
     * Boolean flag indicating that communication with the
     * server is in progress.
     *
     * It is used to disable action buttons when the communication is
     * in progress and enable the buttons when the communication is over.
     */
    busy = false

    /**
     * Boolean flag indicating that loading data is in progress.
     */
    loading = false

    /**
     * Boolean flag indicating if fetching the config reports from the
     * server failed.
     */
    refreshFailed = false

    /**
     * Boolean flag indicating if only reports containing reports should be
     * visible. If false, all reports are shown.
     */
    issuesOnly = true

    /**
     * A total number of reports containing issues.
     */
    totalIssues = 0

    /**
     * A total number of reports.
     */
    totalReports = 0

    /**
     * Component constructor.
     *
     * @param msgService a service used to display error messages.
     * @param servicesApi a service used to fetch the config review reports.
     */
    constructor(
        private msgService: MessageService,
        private servicesApi: ServicesService
    ) {}

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
     * Clears review report information.
     */
    private _resetDefaults() {
        this.start = 0
        this.limit = 5
        this.total = 0
        this.totalIssues = 0
        this.totalReports = 0
        this.reports = []
        this.review = null
        this.busy = false
        this.loading = false
        this.issuesOnly = true
    }

    /**
     * Fetches the configuration reports using pagination and retries.
     *
     * The server performs config reviews in background. If a client makes
     * the request while the review is in progress the server returns
     * HTTP Accepted status code. In this case the function will retry
     * several times with increasing delays between the consecutive
     * attempts. If the server keeps sending the Accepted status code
     * this function will eventually give up and show a warning message
     * to the user. The user will be able to manually refresh the
     * config reports list using a button displayed after the function
     * gives up.
     *
     * @param event an event emitted when user navigates over the pages;
     *        it comprises the offset and limit of reports to fetch.
     * @param useDelay an optional parameter indicating whether the
     *        function should delay subsequent retries to get new config
     *        reports. It should be set to false in the unit tests.
     * @param retries an optional parameter specifying how many times
     *        the function should retry after receiving the Accepted
     *        status code.
     */
    refreshDaemonConfigReports(event, useDelay = true) {
        if (this.loading) {
            return
        }

        const retries = 5
        this.loading = true
        if (event) {
            this.start = event.first ?? this.start
            this.limit = event.rows ?? this.limit
        } else {
            this.start = 0
            this.limit = 5
        }
        // Get reports with specifying the limits.
        this.servicesApi
            .getDaemonConfigReports(this.daemonId, this.start, this.limit, this.issuesOnly, 'response')
            .pipe(
                // Look into the response and extract the status code.
                map((resp) => {
                    // The status code Accepted indicates that the server
                    // is busy generating the review. Since Accepted is a
                    // success response, we need to explicitly throw the
                    // response to pass it to the retryWhen operator below.
                    if (resp.status === HttpStatusCode.Accepted) {
                        throw resp
                    }
                    // Other response types are simply passed through this
                    // operator to trigger the default behavior for them.
                    return resp
                }),
                // retryWhen is invoked when the request fails (an actual error)
                // or when the Accepted status code is returned. We will have to
                // distinguish between these two cases and throw an error if this
                // is an actual error, or retry if we received the Accepted status
                // code.
                retryWhen((errors) =>
                    errors.pipe(
                        // Limit the number of retries. Stop if we're about to exceed
                        // the number of retries. Note that the take() can't follow
                        // the concatMap() because the latter needs to handle the
                        // case when we exceed the retries limit (i.e., show the warning
                        // message). The take(retries+1) will allow for handling this
                        // case and stop afterwards.
                        take(retries + 1),
                        // Look into the error (an actual error or Accepted status).
                        concatMap(
                            // The retryNum is a call index. We use it to track how
                            // many retries were performed so far.
                            (errResp, retryNum) => {
                                // Default behavior for all errors (i.e., stop trying).
                                // Exclude the error resulting from receiving the
                                // Accepted status code.
                                if (errResp.status !== HttpStatusCode.Accepted) {
                                    throw errResp
                                }
                                // If we're about to exceed the number of retries and
                                // we are still receiving the Accepted status we should
                                // give up retrying. Ensure to mark that the refresh
                                // has failed. Also, display the warning message to the
                                // user.
                                if (retryNum >= retries) {
                                    this.busy = false
                                    this.refreshFailed = true

                                    this.msgService.add({
                                        severity: 'warn',
                                        summary: 'Unable to refresh config review reports',
                                        detail:
                                            'Config review is in progress for this daemon. Try refreshing ' +
                                            'the reports later.',
                                        life: 10000,
                                    })
                                } else if (useDelay) {
                                    // If we're still to retry, let's introduce a delay.
                                    // The delay is a function of the retry counter.
                                    // Second retry is after 1s, third retry is after
                                    // 3s since first failure, forth retry is after 7s
                                    // and so on.
                                    return of(errResp).pipe(delay((retryNum + 1) * 1000))
                                }
                                // No delay or exceeded the number of retries.
                                return of(errResp)
                            }
                        )
                    )
                )
            )
            .toPromise()
            .then((resp) => {
                // The resp will be null when we keep getting Accepted status code
                // and the number of retries have been exceeded. We already handled
                // this case so we can simply return from here.
                if (!resp) {
                    return
                }
                // It seems that we successfully fetched the config reports.
                this.refreshFailed = false

                switch (resp.status) {
                    case HttpStatusCode.Ok:
                        this.reports = resp.body.items ?? []
                        this.total = resp.body.total ?? 0
                        this.totalReports = resp.body.totalReports ?? 0
                        this.totalIssues = resp.body.totalIssues ?? 0
                        this.review = resp.body.review
                        this.busy = false
                        break
                    case HttpStatusCode.NoContent:
                        // No review available for this daemon.
                        this._resetDefaults()
                        break
                    default:
                        break
                }
            })
            .catch((err) => {
                let msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error getting review reports',
                    detail: 'Error getting review reports: ' + msg,
                    life: 10000,
                })
                this._resetDefaults()
                this.refreshFailed = true
            })
            .finally(() => {
                this.loading = false
            })
    }

    /**
     * Sends a request to the server to begin a review for a specified daemon.
     *
     * If the request is successful, a request to get the updated configuration
     * reports is subsequently sent. Since the reports are generated in a
     * background, it is possible that the request to get the updated reports
     * returns an Accepted status code indicating that the review is not
     * ready yet. In this case the refreshDaemonConfigReports function retries
     * several times.
     *
     * @param useRefreshDelay an optional parameter indicating whether the
     *        refreshDaemonConfigReports function should delay subsequent
     *        retries to get generated config reports. It should be set to
     *        false in the unit tests.
     */
    runReview(useRefreshDelay = true) {
        this.busy = true
        this.servicesApi
            .putDaemonConfigReview(this.daemonId, 'response')
            .toPromise()
            .then((/* resp */) => {
                this.busy = false
                this.refreshDaemonConfigReports(null, useRefreshDelay)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error running new review',
                    detail: 'Error running new review: ' + msg,
                    life: 10000,
                })
                this.busy = false
            })
    }
}
