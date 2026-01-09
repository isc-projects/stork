import { Component, input, OnInit, output } from '@angular/core'
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
     * Output property emitting events whenever selected daemon changes.
     * It emits daemon ID.
     */
    valueChange = output<number>()

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
                label: `${d.name}@${d.machine.address}`,
            }))
            return
        }

        lastValueFrom(this.servicesApi.getDaemonsDirectory(query, this.domain()))
            .then((response) => {
                const _daemonSuggestions = response.items ?? []
                this.currentSuggestions = _daemonSuggestions.map((d: SimpleDaemon) => ({
                    ...d,
                    label: `${d.name}@${d.machine.address}`,
                }))
            })
            .catch(() => {
                this.errorOccurred.emit('Failed to retrieve daemons from Stork server.')
                this.currentSuggestions = this.daemonSuggestions.map((d: SimpleDaemon) => ({
                    ...d,
                    label: `${d.name}@${d.machine.address}`,
                }))
            })
    }

    /**
     * Callback called whenever selected daemon changes.
     * @param d selected daemon
     */
    onValueChange(d: LabeledSimpleDaemon) {
        console.log('value changed', d)
        this.valueChange.emit(d?.id ?? null)
    }
}
