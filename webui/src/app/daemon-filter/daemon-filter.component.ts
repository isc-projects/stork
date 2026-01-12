import { Component, effect, input, model, OnInit, output } from '@angular/core'
import { AutoComplete, AutoCompleteCompleteEvent } from 'primeng/autocomplete'
import { FloatLabel } from 'primeng/floatlabel'
import { FormsModule } from '@angular/forms'
import { lastValueFrom } from 'rxjs'
import { ServicesService, SimpleDaemon } from '../backend'

type LabeledSimpleDaemon = SimpleDaemon & { label: string }

@Component({
    selector: 'app-daemon-filter',
    imports: [AutoComplete, FloatLabel, FormsModule],
    templateUrl: './daemon-filter.component.html',
    styleUrl: './daemon-filter.component.sass',
})
export class DaemonFilterComponent implements OnInit {
    /**
     * All daemons suggested in the autocomplete component.
     */
    daemonSuggestions: SimpleDaemon[] | undefined

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
     */
    valueChangeEffect = effect(() => {
        const currentValue = this.daemonID()
        const selectedDaemon = this.daemonSuggestions
            ?.map((d: SimpleDaemon) => ({
                ...d,
                label: this.constructDaemonLabel(d),
            }))
            .find((d) => d.id == currentValue)
        this.daemon = selectedDaemon || null
    })

    /**
     * Output property emitting events whenever error occurs while fetching daemons from backend.
     * It may be used to display feedback messages in the parent component.
     */
    errorOccurred = output<string>()

    constructor(private servicesApi: ServicesService) {}

    ngOnInit() {
        lastValueFrom(this.servicesApi.getDaemonsDirectory(undefined, this.domain()))
            .then((response) => {
                this.daemonSuggestions = response.items ?? []
                if (this.daemonID()) {
                    const selectedDaemon = this.daemonSuggestions
                        .map((d: SimpleDaemon) => ({
                            ...d,
                            label: this.constructDaemonLabel(d),
                        }))
                        .find((d) => d.id == this.daemonID())
                    this.daemon = selectedDaemon || null
                }
            })
            .catch(() => {
                this.errorOccurred.emit('Failed to retrieve daemons from Stork server.')
                this.daemonSuggestions = []
            })
    }

    /**
     * Function called to search results for the autocomplete component.
     * @param event
     */
    searchDaemon(event: AutoCompleteCompleteEvent) {
        const query = event.query.trim()
        if (!query) {
            this.currentSuggestions = this.daemonSuggestions.map((d: SimpleDaemon) => ({
                ...d,
                label: this.constructDaemonLabel(d),
            }))
            return
        }

        lastValueFrom(this.servicesApi.getDaemonsDirectory(query, this.domain()))
            .then((response) => {
                const _daemonSuggestions = response.items ?? []
                this.currentSuggestions = _daemonSuggestions.map((d: SimpleDaemon) => ({
                    ...d,
                    label: this.constructDaemonLabel(d),
                }))
            })
            .catch(() => {
                this.errorOccurred.emit('Failed to retrieve daemons from Stork server.')
                this.currentSuggestions = this.daemonSuggestions.map((d: SimpleDaemon) => ({
                    ...d,
                    label: this.constructDaemonLabel(d),
                }))
            })
    }

    /**
     * Callback called whenever selected daemon changes.
     * @param d selected daemon
     */
    onValueChange(d: LabeledSimpleDaemon) {
        console.log('value changed', d)
        this.daemonID.set(d?.id ?? null)
    }

    /**
     * Constructs a label for the daemon.
     * @param d daemon
     * @private
     */
    private constructDaemonLabel(d: SimpleDaemon) : string {
        // TODO: This could be user defined label, once backend supports it.
        // if (d?.machine.label) {
        //     return `${d.name}@${d.machine.label}`
        // }

        if (d?.machine.hostname) {
            return `${d.name}@${d.machine.hostname}`
        }

        if (d?.machine.address) {
            return `${d.name}@${d.machine.address}`
        }

        return `${d.name}@machine ID ${d.machineId}`
    }
}
