import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { Machine, ServicesService } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Location } from '@angular/common'

export interface MachinesFilter {
    text?: string
    authorized?: boolean
}

@Component({
    selector: 'app-authorized-machines-table',
    templateUrl: './authorized-machines-table.component.html',
    styleUrl: './authorized-machines-table.component.sass',
})
export class AuthorizedMachinesTableComponent
    extends PrefilteredTable<MachinesFilter, Machine>
    implements OnInit, OnDestroy
{
    /**
     * Array of all numeric keys that are supported when filtering machines via URL queryParams.
     * Note that it doesn't have to contain machines prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all boolean keys that are supported when filtering machines via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof MachinesFilter)[] = ['authorized']

    /**
     * Array of all numeric keys that can be used to filter machines.
     */
    filterNumericKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all boolean keys that can be used to filter machines.
     */
    filterBooleanKeys: (keyof MachinesFilter)[] = ['authorized']

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'authorized-machines-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof MachinesFilter = 'authorized'

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values.
     */
    filterValidators = []

    /**
     * PrimeNG table instance.
     */
    @ViewChild('machinesTable') table: Table

    @Output() machineMenuDisplay: EventEmitter<Machine> = new EventEmitter()

    @Output() authorizeMachines: EventEmitter<Machine[]> = new EventEmitter()

    selectedMachines: Machine[] = []

    // This counter is used to indicate in UI that there are some
    // unauthorized machines that may require authorization.
    @Input() unauthorizedMachinesCount = 0

    /**
     *
     * @param machine
     */
    onMachineMenuDisplay(machine: Machine) {
        this.machineMenuDisplay.emit(machine)
    }

    /**
     *
     * @param machines
     */
    onAuthorizeMachines(machines: Machine[]) : void {
        console.log("onAuthorizeMachines", machines)
        this.authorizeMachines.emit(machines)
    }

    // machines: Machine[] = []

    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService,
        private messageService: MessageService,
        private location: Location
    ) {
        super(route, location)
    }

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
    }
    loadData(event: TableLazyLoadEvent): void {
        // Indicate that machines refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getMachines(JSON.stringify(event))
        lastValueFrom(
            this.servicesApi.getMachines(
                event.first,
                event.rows,
                this.getTableFilterValue('text', event.filters),
                null,
                this.prefilterValue ?? this.getTableFilterValue('authorized', event.filters)
            )
        )
            .then((data) => {
                this.dataCollection = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get machine list',
                    detail: 'Error getting machine list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    authorizedMachinesDisplayed(): boolean {
        return this.dataCollection?.some((m) => m.authorized) || false
    }

    unauthorizedMachinesDisplayed(): boolean {
        return this.dataCollection?.some((m) => !m.authorized) || false
    }
}
