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
    subscription: Subscription = null

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
     * It fetches a list of apps with communication issues. When this
     * list is returned a warning message may be displayed. Then, it
     * subscribes to the events related to the connectivity issues to
     * track any new alerts of that kind.
     */
    ngOnInit(): void {
        this.getAppsWithCommunicationIssues()
    }

    /**
     * A lifecycle hook invoked when the component is destroyed.
     *
     * It unsubscribes from the events service.
     */
    ngOnDestroy(): void {
        if (this.subscription) {
            this.subscription.unsubscribe()
        }
    }

    /**
     * Inserts new message under the specified key.
     *
     * If the message with this key (stream name) already exists, it is
     * replaced.
     *
     * @param key message key.
     * @param message message to be displayed.
     */
    private insertMessage(key: string, message: Message): void {
        const index = this.messages.findIndex((message) => message.key === key)
        if (index >= 0) {
            // The message under this key already exists. Replace it.
            this.messages[index] = message
        } else {
            // The message for this key does not exist. Insert it at the
            // top of all messages.
            this.messages.unshift(message)
        }
        // Shallow copy the array. Assigning new value to the messages is
        // important to trigger view change detection. If the array is
        // empty the call to slice(0) returns an empty array.
        this.messages = this.messages.slice(0)
    }

    /**
     * Deletes message by key.
     *
     * If the message doesn't exist it does nothing.
     *
     * @param key message key.
     */
    private deleteMessage(key: string): void {
        const index = this.messages.findIndex((message) => message.key === key)
        if (index >= 0) {
            this.messages.splice(index, 1)
        }
        // Shallow copy the array to trigger view change detection.
        this.messages = this.messages.slice(0)
    }

    /**
     * Get the list of apps reporting communication issues from the server.
     *
     * If there is at least one such app a warning message is displayed.
     * Otherwise the message is deleted. To prevent many consecutive calls
     * to this function it sets a backoff mechanism with a timeout. It will
     * be called after the timeout elapses if there have been any events
     * captured during the backoff.
     *
     * When it gets the list of issues for the first time it subscribes to
     * the events related to the connectivity issues.
     */
    private getAppsWithCommunicationIssues(): void {
        lastValueFrom(this.servicesApi.getAppsWithCommunicationIssues())
            .then((data) => {
                if (data.total > 0) {
                    const message: Message = {
                        key: 'connectivity',
                        severity: 'warn',
                        summary: 'Communication issues',
                        detail:
                            `Stork server reports communication problems for ${formatNoun(data.total, 'app', 's')} ` +
                            `on the monitored machines. You can check the details <a href="/communication">here</a>.`,
                    }
                    this.insertMessage('connectivity', message)
                } else {
                    this.deleteMessage('connectivity')
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
                if (!this.subscription) {
                    this.subscribe()
                } else {
                    // Use a backoff mechanism with a timeout.
                    this.backoff = true
                    this.setBackoffTimeout()
                }
            })
    }

    /**
     * Subscribes to the events indicating the connectivity issues and the
     * control events occuring on SSE reconnection.
     */
    private subscribe(): void {
        this.subscription = this.sse
            .receivePriorityEvents()
            .pipe(
                filter(
                    (event) =>
                        event.stream === 'all' || event.stream === 'connectivity' || event.stream === 'registration'
                )
            )
            .subscribe((event) => {
                if (event.stream === 'all' || event.stream === 'connectivity') {
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
                } else if (event.stream === 'registration') {
                    const message: Message = {
                        key: 'registration',
                        severity: 'warn',
                        summary: 'New registration request',
                        detail:
                            `There are new machines requesting registration and awaiting approval. ` +
                            `Visit the list of unauthorized machines <a href="/machines/all">here</a>.`,
                    }
                    this.insertMessage('registration', message)
                }
            })
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
