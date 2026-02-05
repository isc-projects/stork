import { Component, effect, input, model, OnDestroy, OnInit, output } from '@angular/core'
import { AutoComplete, AutoCompleteCompleteEvent } from 'primeng/autocomplete'
import { FloatLabel } from 'primeng/floatlabel'
import { FormsModule } from '@angular/forms'
import {
    exhaustMap,
    Observable,
    of,
    Subject,
    Subscription,
    switchMap,
    tap,
    throttleTime,
    throwError,
    timeout,
} from 'rxjs'
import { ServicesService, SimpleDaemon, SimpleDaemons } from '../backend'
import { getErrorMessage } from '../utils'
import { catchError, concatMap, share } from 'rxjs/operators'

/**
 * Type extending SimpleDaemon with a string label.
 */
type LabeledSimpleDaemon = SimpleDaemon & { label: string }

/**
 * This component provides PrimeNG Autocomplete form element with a list
 * of all DHCP and DNS daemons known to Stork server.
 * It supports either selecting the daemon from a dropdown list or
 * searching the daemon by name, machines hostname, machines address.
 * Selected daemon ID can be accessed via daemonID input/output property.
 * Parent component may also inject the ID so the
 * component will display it as selected.
 */
@Component({
    selector: 'app-daemon-filter',
    imports: [AutoComplete, FloatLabel, FormsModule],
    templateUrl: './daemon-filter.component.html',
    styleUrl: './daemon-filter.component.sass',
})
export class DaemonFilterComponent implements OnInit, OnDestroy {
    /**
     * All daemons suggested in the autocomplete component.
     * @private
     */
    private daemonSuggestions: LabeledSimpleDaemon[] = []

    /**
     * Daemons suggestions matching current autocomplete query.
     */
    currentSuggestions: LabeledSimpleDaemon[] | undefined

    /**
     * Currently selected daemon.
     */
    daemon: LabeledSimpleDaemon | undefined

    /**
     * Input property controlling whether to display daemons from DHCP, DNS domains or both.
     * If undefined, all domains are used.
     */
    domain = input<'dns' | 'dhcp' | undefined>(undefined)

    /**
     * Input property with label value.
     */
    label = input<string>('Daemon (type or pick)')

    /**
     * Input/Output (ModelSignal) property emitting daemon ID whenever selected daemon changes.
     * It also accepts input daemonID to update selected daemon in the autocomplete component.
     */
    daemonID = model<number>(undefined)

    /**
     * Effect reacting on daemonID change done by parent component. It updates model of the autocomplete component.
     * @private
     */
    private valueChangeEffect = effect(() => {
        const currentValue = this.daemonID()
        const selectedDaemon = this.daemonSuggestions.find((d) => d.id == currentValue)
        this.daemon = selectedDaemon || null
    })

    /**
     * Output property emitting events whenever error occurs while fetching daemons from backend.
     * It may be used to display feedback messages in the parent component.
     */
    errorOccurred = output<string>()

    /**
     * List of daemon types that will be visualized by this component.
     * It is used to filter out Kea daemons like control-agent or ddns which are not relevant for this component use.
     * @private
     */
    private acceptedDaemons = ['dhcp4', 'dhcp6', 'named', 'pdns']

    /**
     * RxJS subject triggering the getDaemonsDirectory API call.
     * @private
     */
    private callApiTrigger = new Subject()

    /**
     * Timeout value in milliseconds stating how long does the component wait for backend response
     * until error feedback message is emitted.
     * @private
     */
    private timeoutValue = 2500

    /**
     * RxJS stream emitting daemons directory. It is calling REST API getDaemonsDirectory
     * in a protected, throttled way to mitigate the backend load. In a successful scenario,
     * first attempt of retrieving the daemons directory will be the only one.
     * It is also secured by a timeout, in case there is no response from REST API for long time.
     * This is to prevent frozen UI experience.
     * @private
     */
    private receivedDaemons$: Observable<SimpleDaemons> = this.callApiTrigger.pipe(
        throttleTime(3000),
        exhaustMap(() =>
            this.servicesApi.getDaemonsDirectory(undefined, this.domain()).pipe(
                timeout({
                    each: this.timeoutValue,
                    with: () => throwError(() => `timeout - no response in ${this.timeoutValue}ms`),
                }),
                catchError((err) => {
                    const msg = getErrorMessage(err)
                    this.errorOccurred.emit(`Failed to retrieve daemons from Stork server: ${msg}`)
                    return of({ items: [] })
                })
            )
        ),
        share()
    )

    /**
     * RxJS subscription keeping all subscriptions so that they can be all unsubscribed once the component gets destroyed.
     * @private
     */
    private subscription: Subscription

    /**
     * Component ctor.
     * @param servicesApi services API used to retrieve daemons directory from backend
     */
    constructor(private servicesApi: ServicesService) {}

    /**
     * Initializes the component. It subscribes to receivedDaemons$ RxJS stream
     * and triggers the first attempt of retrieving daemons directory.
     */
    ngOnInit() {
        this.subscription = this.receivedDaemons$.subscribe((data) => {
            this.daemonSuggestions = (data.items?.filter((d) => this.acceptedDaemons.includes(d.name)) ?? []).map(
                (d) => ({
                    ...d,
                    label: this.constructDaemonLabel(d),
                })
            )
            if (this.daemonID()) {
                const selectedDaemon = this.daemonSuggestions.find((d) => d.id == this.daemonID())
                this.daemon = selectedDaemon || null
            }
        })

        this.subscription.add(
            this.daemonLookup$.subscribe((query) => {
                this.daemonLookup(query)
            })
        )

        this.callGetDaemonsAPI()
    }

    /**
     * Does the cleanup once the component gets destroyed.
     * It unsubscribes from RxJS observables and completes RxJS subject.
     */
    ngOnDestroy() {
        this.subscription.unsubscribe()
        this.callApiTrigger.complete()
        this.searchDaemonTrigger.complete()
    }

    /**
     * Calls Stork server REST API to retrieve all daemons that are known to the server.
     * @private
     */
    private callGetDaemonsAPI() {
        this.callApiTrigger.next(null)
    }

    /**
     * RxJS subject emitting query string whenever value is typed into autocomplete form,
     * or the dropdown button is clicked.
     * @private
     */
    private searchDaemonTrigger = new Subject<string>()

    /**
     * RxJS stream emitting query string from the autocomplete form in a coordinated way.
     * If user typed a query (or clicked on a dropdown button), but the daemons directory
     * was not yet received from REST API, it will first wait for the response, and then will emit
     * the query.
     * If user changes the query in the meantime, before API response arrives, only the last query
     * typed by user will be emitted.
     * This RxJS stream is designed for deployments where responses from REST API may be slower.
     * @private
     */
    private daemonLookup$ = this.searchDaemonTrigger.pipe(
        tap(() => this.callGetDaemonsAPI()),
        switchMap((s) => this.receivedDaemons$.pipe(concatMap(() => of(s))))
    )

    /**
     * Function called to search results for the autocomplete component.
     * @param event autocomplete event with the query
     */
    searchDaemon(event: AutoCompleteCompleteEvent) {
        if (this.daemonSuggestions.length) {
            this.daemonLookup(event.query)
            return
        }

        this.searchDaemonTrigger.next(event.query)
    }

    /**
     * Function called to perform daemon lookup based on given key string.
     *
     * @param key keyword used for daemons lookup
     * @private
     */
    private daemonLookup(key: string) {
        const query = key.trim()
        if (!query) {
            this.currentSuggestions = [...this.daemonSuggestions]
            return
        }

        // There are two ways to do daemons lookup:
        // 1. on backend side - REST API supports searching daemons by query string
        // 2. on frontend side - all daemons should be in daemonSuggestions, so the filtering can be done on daemonSuggestions
        // Let's try second approach for now (less load for the backend).
        const filtered = this.daemonSuggestions.filter(
            (d) => d.name.includes(query) || d.machine?.hostname?.includes(query) || d.machine?.address?.includes(query)
        )
        this.currentSuggestions = [...filtered]

        // Comment out filtering on backend side.
        // lastValueFrom(this.servicesApi.getDaemonsDirectory(query, this.domain()))
        //     .then((response) => {
        //         const _daemonSuggestions = response.items.filter((d) => this.acceptedDaemons.includes(d.name)) ?? []
        //         this.currentSuggestions = _daemonSuggestions.map((d: SimpleDaemon) => ({
        //             ...d,
        //             label: this.constructDaemonLabel(d),
        //         }))
        //     })
        //     .catch(() => {
        //         this.errorOccurred.emit('Failed to retrieve daemons from Stork server.')
        //         this.currentSuggestions = this.daemonSuggestions?.map((d: SimpleDaemon) => ({
        //             ...d,
        //             label: this.constructDaemonLabel(d),
        //         }))
        //     })
    }

    /**
     * Callback called whenever selected daemon changes.
     * @param d selected daemon
     */
    onValueChange(d: LabeledSimpleDaemon) {
        this.daemonID.set(d?.id ?? null)
    }

    /**
     * Constructs a label for the daemon.
     * @param d daemon
     * @private
     */
    private constructDaemonLabel(d: SimpleDaemon): string {
        // TODO: This could be user defined label, once backend supports it.
        // if (d.machine?.label) {
        //     return `${d.name}@${d.machine.label}`
        // }

        if (d.machine?.hostname) {
            return `${d.name}@${d.machine.hostname}`
        }

        if (d.machine?.address) {
            return `${d.name}@${d.machine.address}`
        }

        return `${d.name}@machine ID ${d.machineId}`
    }
}
