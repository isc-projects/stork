import { Component, OnDestroy, OnInit } from '@angular/core'
import { ServicesService } from '../backend'
import { Message, MessageService } from 'primeng/api'
import { ServerSentEventsService } from '../server-sent-events.service'
import { Subscription, filter, lastValueFrom } from 'rxjs'
import { formatNoun, getErrorMessage } from '../utils'

/**
 * A panel alerting about communication problems.
 *
 * This panel is displayed just below the main menu. It subscribes to the
 * events related to the communication issues between the Stork server and
 * the monitored systems. If it detects a communication issue it displays
 * a well visible alert.
 */
@Component({
    selector: 'app-priority-errors-panel',
    templateUrl: './priority-errors-panel.component.html',
    styleUrls: ['./priority-errors-panel.component.sass'],
})
export class PriorityErrorsPanelComponent implements OnInit, OnDestroy {
    /**
     * Holds displayed alerts.
     */
    messages: Message[] = []

    /**
     * A subscription to the SSE service receiving the events.
     */
    subscription: Subscription

    /**
     * Is the backoff mechanism enabled.
     *
     * When this flag is true, no new requests to the server are issued
     * to fetch the apps with the connectivity problems.
     */
    backoff = false

    /**
     * Counts the events received during the backoff.
     */
    eventCount = 0

    /**
     * Constructor.
     *
     * @param messageService message service
     * @param sse server sent events service
     * @param servicesApi REST API service
     */
    constructor(
        private messageService: MessageService,
        private sse: ServerSentEventsService,
        private servicesApi: ServicesService
    ) {}

    /**
     * A lifecycle hook invoked when the component is initialized.
     *
     * It subscribes to the events related to the connectivity issues.
     * When it receives such an event, it makes a call to the server to
     * fetch a detailed information about the communication issues.
     */
    ngOnInit(): void {
        this.subscription = this.sse
            .receiveConnectivityEvents()
            .pipe(filter((event) => event.stream === 'all' || event.stream === 'connectivity'))
            .subscribe(() => {
                // To avoid many subsequent calls getting the current communication state,
                // we introduce a backoff mechanism. After getting the detailed communication
                // state we set a timeout when we only get notified about the events but not
                // actually query the server for communication issues.
                if (this.backoff) {
                    this.eventCount++
                } else {
                    // When we receive such an event something has potentially changed in the
                    // status of the connectivity between the server and the machines. Let's
                    // get the details.
                    this.getAppsWithCommunicationIssues()
                }
            })
    }

    /**
     * Get the list of apps reporting communication issues from the server.
     *
     * If there is at least one such app a warning message is displayed.
     * Otherwise the message is deleted. To prevent many consecutive calls
     * to this function it sets a backoff mechanism with a timeout. It will
     * be called after the timeout elapses if there have been any events
     * captured during the backoff.
     */
    private getAppsWithCommunicationIssues(): void {
        lastValueFrom(this.servicesApi.getAppsWithCommunicationIssues())
            .then((data) => {
                const apps = data.items ?? []
                if (apps.length > 0) {
                    this.messages = [
                        {
                            severity: 'warn',
                            summary: 'Communication issues',
                            detail:
                                `Stork server reports communication problems for ${formatNoun(
                                    apps.length,
                                    'app',
                                    's'
                                )} ` +
                                `on the monitored machines. Please check if the apps and the Stork agents are running.`,
                        },
                    ]
                } else {
                    this.messages = []
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get communication issues report',
                    detail: `Error getting communication issues report: ${msg}`,
                    life: 10000,
                })
            })
            .finally(() => {
                // Use a backoff mechanism with a timeout.
                this.backoff = true
                this.setBackoffTimeout()
            })
    }

    /**
     * A lifecycle hook invoked when the component is destroyed.
     *
     * It unsubscribes from the events service.
     */
    ngOnDestroy(): void {
        this.subscription.unsubscribe()
    }

    /**
     * Enables the backoff mechanism.
     *
     * It sets a timeout until next attempt to fetch the apps can be made.
     * When the timeout elapses, it fetches an updated list of apps from the
     * server if any new events have come during the backoff time.
     */
    setBackoffTimeout(): void {
        setTimeout(() => {
            // The timeout elapsed. We can now clear the backoff flag
            // and issue another request if any events have been captured.
            this.backoff = false
            if (this.eventCount > 0) {
                this.eventCount = 0
                this.getAppsWithCommunicationIssues()
            }
        }, 5000)
    }
}
