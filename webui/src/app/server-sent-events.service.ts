import { Injectable } from '@angular/core'
import { BehaviorSubject, Observable } from 'rxjs'
import { Event } from './backend'

/**
 * A filter for the server sent events.
 */
export interface SSEFilter {
    level?: number
    machine?: number
    appType?: string
    daemonType?: string
    user?: string
}

/**
 * A wrapper for the received event.
 *
 * It contains an SSE stream identifier and the original event
 * received from the Stork server.
 */
export interface SSEEvent {
    stream: string
    originalEvent: Event | null
}

/**
 * Server Sent Events (SSE) Service.
 *
 * SSE is a popular mechanism for subscribing to and receiving notifications
 * from the server about interesting events. A client opens a connection to
 * the server and the server returns events in a well defined format as they
 * occur.
 *
 * The SSE can subscriptions can be requested in different components but it
 * is not practical to open individual connections at the same time. The browsers
 * have a limit on the number of concurrent connections, so it makes sene to
 * aggregate all events in a single connection to avoid exceeding this limit.
 *
 * This service aggregates the SSE subscriptions in a single connection and
 * make the received events available to the components via an observable.
 * A change in filtering causes the service to re-establish the SSE
 * connection.
 */
@Injectable({
    providedIn: 'root',
})
export class ServerSentEventsService {
    /**
     * SSE connection error counter.
     *
     * It is reset upon successful reception of an event and increased on
     * error. High number of errors delays an attempt to reconnect to the
     * server to avoid the reconnection storm.
     */
    private errorCount = 0

    /**
     * A source of events from the server.
     *
     * It is used to establish SSE connection with the server, receive the
     * events and handle communication errors.
     */
    private eventSource: EventSource | null

    /**
     * An observable returned to the subscribing components.
     *
     * This service pushes the events received over SSE to this observable
     * and the components receive the events over it.
     */
    private events$: Observable<SSEEvent>

    /**
     * A behavior subject used to push the new events to the observable.
     */
    private receivedEventsSubject: BehaviorSubject<SSEEvent>

    /**
     * Holds currently established subscriptions to SSE streams.
     *
     * The key holds a stream name. The value holds an applied filter.
     */
    private subscriptions: Map<string, SSEFilter> = new Map()

    /**
     * Constructor.
     *
     * Creates an observable for the components to subscribe.
     */
    constructor() {
        this.receivedEventsSubject = new BehaviorSubject<SSEEvent>({
            stream: 'all',
            originalEvent: null,
        })
        this.events$ = this.receivedEventsSubject.asObservable()
    }

    /**
     * Subscribe to a stream of the connectivity events in the server.
     *
     * The connectivity events report issues with the connectivity between
     * the server and the monitored machines. The components reporting such
     * issues should call this function to subscribe to these events.
     *
     * If the SSE connection has been already established and it includes the
     * subscription to such events this function does not re-open the connection.
     * Note that other functions may subscribe to the connectivity events
     * besides other streams.
     *
     * @returns an observable providing the events from the SSE stream.
     */
    public receiveConnectivityEvents(): Observable<SSEEvent> {
        if (!this.subscriptions.has('connectivity')) {
            let subscription: SSEFilter = {
                level: 0,
            }
            this.subscriptions.set('connectivity', subscription)
            this.reopenSSEConnection()
        }
        return this.events$
    }

    /**
     * Subscribe to a connectivity and message (default) streams of events in the server.
     *
     * The message events are typically displayed in the events panel component. However
     * this function also subscribes to the connectivity events which are monitored on
     * each Stork page. Subscribing to both streams at once makes sense assuming that the
     * subscription to the connectivity events is almost always required.
     *
     * @param filter a filter for the message events (e.g., by machine ID).
     * @returns an observable providing the events from the SSE stream.
     */
    public receiveConnectivityAndMessageEvents(filter: SSEFilter): Observable<SSEEvent> {
        // See if we need to reconnect.
        if (this.subscriptions.has('connectivity') && this.subscriptions.has('message')) {
            let existingSubscription = this.subscriptions.get('message')
            if (JSON.stringify(existingSubscription) === JSON.stringify(filter)) {
                // We already have matching subscription. Let's just return.
                return this.events$
            }
        }
        // Need to re-establish SSE connection because our filtering parameters
        // have changed or we haven't subscribed to some streams yet.
        this.subscriptions.set('connectivity', {})
        this.subscriptions.set('message', filter)
        this.reopenSSEConnection()
        return this.events$
    }

    /**
     * Establishes the SSE connection to the server.
     *
     * If the connection already exists, it is closed and a new connection
     * is established.
     */
    private reopenSSEConnection() {
        // Build event source URL from the filters.
        const searchParams = new URLSearchParams()
        // The message subscription can have a bunch of filters, so it needs
        // special handling.
        const messageSubscription = this.subscriptions.get('message')
        if (messageSubscription) {
            if (messageSubscription.machine) {
                searchParams.append('machine', String(messageSubscription.machine))
            }
            if (messageSubscription.appType) {
                searchParams.append('appType', messageSubscription.appType)
            }
            if (messageSubscription.daemonType) {
                searchParams.append('daemonName', messageSubscription.daemonType)
            }
            if (messageSubscription.user) {
                searchParams.append('user', String(messageSubscription.user))
            }
            if (messageSubscription.level) {
                searchParams.append('level', String(messageSubscription.level))
            }
        }
        // Finally, add the streams to the subscription.
        this.subscriptions.forEach((_, key) => {
            searchParams.append('stream', key)
        })

        // Ensure that the connection is closed before we re-open it.
        this.closeEventSource()
        this.addEventListeners(`/sse?${searchParams.toString()}`)
    }

    /**
     * Start the listeners for each subscription and for errors.
     *
     * @param url an url the listener should connect to
     */
    addEventListeners(url: string) {
        this.eventSource = new EventSource(url)

        // Add an error handler.
        this.eventSource.addEventListener(
            'error',
            () => {
                this.closeEventSource()
                // Indicate to the components that the connection was lost. The
                // stream name all addresses it to all components.
                this.receivedEventsSubject.next({
                    stream: 'all',
                    originalEvent: null,
                })
                // Schedule reconnection. We don't want to re-connect right away
                // to avoid the re-connection storm in case of some persistent
                // problem.
                setTimeout(
                    () => {
                        this.reopenSSEConnection()
                    },
                    // Use a backoff mechanism in case of a recurring error.
                    this.errorCount++ < 10 ? 10000 : 600000
                )
            },
            false
        )

        // We need to install different listeners for different streams,
        // so they can be dispatch to their respective components.
        this.subscriptions.forEach((_, stream) => {
            if (this.eventSource) {
                this.eventSource.addEventListener(
                    stream,
                    (ev) => {
                        const data = JSON.parse(ev.data)
                        this.receivedEventsSubject.next({
                            stream: stream,
                            originalEvent: data,
                        })
                        this.errorCount = 0
                    },
                    false
                )
            }
        })
    }

    /**
     * Closes SSE connection.
     */
    closeEventSource() {
        if (this.eventSource) {
            this.eventSource.close()
            this.eventSource = null
        }
    }
}

/**
 * Provider for a testable ServerSentEventsService.
 *
 * It mocks the function adding listeners, so that the tests don't try
 * to establish actual connections.
 */
@Injectable()
export class TestableServerSentEventsService extends ServerSentEventsService {
    /**
     * Stub function for adding the event listeners used in testing.
     */
    override addEventListeners() {
        console.info('THIS FUNCTION CALLED')
    }
}
