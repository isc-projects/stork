import { Component, OnDestroy, OnInit } from '@angular/core'
import { ServicesService } from '../backend'
import { Message, MessageService } from 'primeng/api'
import { EventStream, ServerSentEventsService } from '../server-sent-events.service'
import { Subscription, filter, lastValueFrom, map } from 'rxjs'
import { formatNoun, getErrorMessage } from '../utils'

/**
 * A panel displaying important messages using the events system.
 *
 * This panel is displayed just below the main menu. It subscribes to the
 * events related to the communication issues between the Stork server and
 * the monitored systems, and to the events informing about new machine
 * registration attempts.
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
     * A backoff mechanism state for different streams.
     *
     * When events are received by the component via one of the streams, it
     * can send requests to the server to pull the detailed information useful
     * for alerting a user. To avoid sending the requests after each received
     * event the component uses a backoff mechanism for each stream. When
     * enabled (i.e., value in this map set to true), the component only
     * counts received events over the stream but doesn't send the requests
     * to the server. The request is sent when a timeout elapses and when
     * some events have been received over the given stream during the backoff.
     */
    private backoff: Map<EventStream, boolean> = new Map()

    /**
     * Counts the events received during the backoff.
     */
    private eventCount: Map<EventStream, number> = new Map()

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
     * It fetches the detailed information about the current communication
     * issues and the number of unauthorized machines. Next, it subscribes
     * to the events related to the communication issues and the new machine
     * registration attempts.
     */
    ngOnInit(): void {
        // Send the requests to the server to check the current situation.
        // Subscribe to the events when we're done with all the requests.
        Promise.all([this.getAppsWithCommunicationIssues(), this.getUnauthorizedMachinesCount()]).then(() => {
            this.subscribe()
        })
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
     * Otherwise, the message is deleted. To prevent many consecutive calls
     * to this function it sets a backoff mechanism with a timeout. It will
     * be called after the timeout elapses if there have been any events
     * captured during the backoff.
     *
     * @returns A promise resolved when the communication completes.
     */
    async getAppsWithCommunicationIssues(): Promise<void> {
        try {
            try {
                const data = await lastValueFrom(this.servicesApi.getAppsWithCommunicationIssues())
                if (data.total > 0) {
                    const message: Message = {
                        key: EventStream.Connectivity,
                        severity: 'warn',
                        summary: 'Communication issues',
                        detail:
                            `Stork server reports communication problems for ${formatNoun(data.total, 'app', 's')} ` +
                            `on the monitored machines. You can check the details <a href="/communication">here</a>.`,
                    }
                    this.insertMessage(EventStream.Connectivity, message)
                } else {
                    this.deleteMessage(EventStream.Connectivity)
                }
            } catch (err) {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get communication issues report',
                    detail: `Error getting communication issues report: ${msg}`,
                    life: 10000,
                })
            }
        } finally {
            // Use a backoff mechanism with a timeout.
            this.setBackoff(EventStream.Connectivity, true)
            this.setBackoffTimeout(EventStream.Connectivity, () => this.getAppsWithCommunicationIssues())
        }
    }

    /**
     * Get the number of the unauthorized machines from the server.
     *
     * If there is at least one unauthorized machine a warning message is
     * displayed. Otherwise, the message is deleted. To prevent many consecutive
     * calls to this function it sets a backoff mechanism with a timeout. It will
     * be called after the timeout elapses if there have been any events
     * captured during the backoff.
     *
     * @returns A promise resolved when the communication completes.
     */
    async getUnauthorizedMachinesCount(): Promise<void> {
        try {
            try {
                const count = await lastValueFrom(this.servicesApi.getUnauthorizedMachinesCount())
                if (count > 0) {
                    const message: Message = {
                        key: EventStream.Registration,
                        severity: 'warn',
                        summary: 'Unregistered machines',
                        detail:
                            `Found ${formatNoun(count, 'machine', 's')} requesting registration and awaiting approval. ` +
                            `Visit the list of <a href="/machines/all?authorized=false">unauthorized machines</a> to review the requests.`,
                    }
                    this.insertMessage(EventStream.Registration, message)
                } else {
                    this.deleteMessage(EventStream.Registration)
                }
            } catch (err) {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get unregistered machines count',
                    detail: `Error getting unregistered machines count: ${msg}`,
                    life: 10000,
                })
            }
        } finally {
            // Use a backoff mechanism with a timeout.
            this.setBackoff(EventStream.Registration, true)
            this.setBackoffTimeout(EventStream.Registration, () => this.getUnauthorizedMachinesCount())
        }
    }

    /**
     * Subscribes to the events indicating the connectivity issues, new machine
     * registration attempts, and the control events occurring on SSE reconnection.
     */
    private subscribe(): void {
        this.subscription = this.sse
            .receivePriorityEvents()
            .pipe(
                map((event) => event.stream as EventStream),
                filter(
                    (stream) =>
                        stream === EventStream.All ||
                        stream === EventStream.Connectivity ||
                        stream === EventStream.Registration
                )
            )
            .subscribe((stream) => {
                if (stream === EventStream.All || stream === EventStream.Connectivity) {
                    this.runConditionally(EventStream.Connectivity, () => this.getAppsWithCommunicationIssues())
                }
                if (stream === EventStream.All || stream === EventStream.Registration) {
                    this.runConditionally(EventStream.Registration, () => this.getUnauthorizedMachinesCount())
                }
            })
    }

    /**
     * Enables or disables backoff for a particular stream.
     *
     * @param eventStream event stream.
     * @param enable a boolean value indicating if the backoff should be
     *        enabled or disabled.
     */
    setBackoff(eventStream: EventStream, enable: boolean): void {
        this.backoff.set(eventStream, enable)
    }

    /**
     * Checks if the backoff has been enabled for a stream.
     *
     * @param eventStream event stream.
     * @returns true if the backoff has been enabled.
     */
    isBackoff(eventStream: EventStream): boolean {
        return !!this.backoff.get(eventStream)
    }

    /**
     * Increases a received events count during the backoff.
     *
     * @param eventStream event stream for which the counter is increased.
     */
    private increaseEventCount(eventStream: EventStream) {
        let count = this.getEventCount(eventStream)
        this.eventCount.set(eventStream, count + 1)
    }

    /**
     * Resets the backoff event count for a stream.
     *
     * @param eventStream event stream.
     */
    resetEventCount(eventStream: EventStream): void {
        this.eventCount.set(eventStream, 0)
    }

    /**
     * Returns the current event count for a stream.
     *
     * @param eventStream event stream.
     * @returns A number of events received so far during the backoff.
     */
    getEventCount(eventStream: EventStream): number {
        return this.eventCount.get(eventStream) ?? 0
    }

    /**
     * Enables the backoff mechanism.
     *
     * It sets a timeout until the next attempt to communicate with the server to
     * fetch the information about the particular issue. When the timeout elapses,
     * it calls the specified callback function to communicate.
     *
     * @param eventStream event stream.
     * @param callback a callback function invoked after the timeout. The callback
     *        typically sends a request to the server to fetch updated information
     *        about the issues communicated by this component.
     */
    setBackoffTimeout(eventStream: EventStream, callback: () => void): void {
        setTimeout(() => {
            // The timeout elapsed. We can now clear the backoff flag
            // and issue another request if any events have been captured.
            this.setBackoff(eventStream, false)
            if (this.getEventCount(eventStream) > 0) {
                this.resetEventCount(eventStream)
                callback()
            }
        }, 5000)
    }

    /**
     * Runs the callback when backoff is disabled or records an event otherwise.
     *
     * @param eventStream event stream.
     * @param callback a callback function to be invoked when the backoff is disabled.
     */
    private runConditionally(eventStream: EventStream, callback: () => void): void {
        if (this.isBackoff(eventStream)) {
            // We have received an event but the backoff is enabled. Record the
            // event but don't send the request to the server.
            this.increaseEventCount(eventStream)
        } else {
            // No backoff, so let's communicate with the server.
            callback()
        }
    }
}
