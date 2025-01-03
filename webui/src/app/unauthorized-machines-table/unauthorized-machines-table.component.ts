import { Component, EventEmitter, OnDestroy, OnInit, Output, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { Machine, ServicesService } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Location } from '@angular/common'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { MachinesFilter } from '../authorized-machines-table/authorized-machines-table.component'

@Component({
    selector: 'app-unauthorized-machines-table',
    templateUrl: './unauthorized-machines-table.component.html',
    styleUrl: './unauthorized-machines-table.component.sass',
})
export class UnauthorizedMachinesTableComponent
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
    queryParamBooleanKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter machines.
     */
    filterNumericKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all boolean keys that can be used to filter machines.
     */
    filterBooleanKeys: (keyof MachinesFilter)[] = []

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'unauthorized-machines-table-session'

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

    machineMenuDisplayed(machine: Machine) {
        this.machineMenuDisplay.emit(machine)
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
                this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                false
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
}
