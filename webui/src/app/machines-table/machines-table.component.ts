import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { Machine, ServicesService } from '../backend'
import { Table, TableLazyLoadEvent, TableSelectAllChangeEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Location } from '@angular/common'

/**
 * Interface defining fields for Machines filter.
 */
export interface MachinesFilter {
    text?: string
    authorized?: boolean
}

/**
 * This component is dedicated to display the table of Machines. It supports
 * lazy data loading from the backend and machines filtering.
 */
@Component({
    selector: 'app-machines-table',
    templateUrl: './machines-table.component.html',
    styleUrl: './machines-table.component.sass',
})
export class MachinesTableComponent extends PrefilteredTable<MachinesFilter, Machine> implements OnInit, OnDestroy {
    /**
     * Array of all numeric keys that are supported when filtering machines via URL queryParams.
     */
    queryParamNumericKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all boolean keys that are supported when filtering machines via URL queryParams.
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
    stateKeyPrefix: string = 'machines-table-session'

    /**
     * queryParam keyword of the prefilter.
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

    /**
     * Output property emitting events to parent component when Show machine's menu button was clicked by user.
     */
    @Output() machineMenuDisplay = new EventEmitter<{ e: Event; m: Machine }>()

    /**
     * Output property emitting events to parent component when Authorize selected machines button was clicked by user.
     */
    @Output() authorizeSelectedMachines = new EventEmitter<Machine[]>()

    /**
     * Array of selected machines.
     */
    selectedMachines: Machine[] = []

    /**
     * This counter is used to indicate in UI that there are some
     * unauthorized machines that may require authorization.
     */
    @Input() unauthorizedMachinesCount = 0

    /**
     * Output property emitting events to parent component when unauthorizedMachinesCount changes.
     */
    @Output() unauthorizedMachinesCountChange = new EventEmitter<number>()

    /**
     * Keeps state of the Select All checkbox in the table's header.
     */
    selectAll: boolean = false

    /**
     * Keeps count of unauthorized machines in the Machines data collection returned from backend.
     * @private
     */
    private _unauthorizedInDataCollectionCount: number = 0

    /**
     * Callback called when Show machine's menu button was clicked by user.
     * @param event browser's click event
     * @param machine machine for which the menu is about to be displayed
     */
    onMachineMenuDisplayClicked(event: Event, machine: Machine) {
        this.machineMenuDisplay.emit({ e: event, m: machine })
    }

    /**
     * Callback called when the Authorize selected machines button was clicked.
     * @param machines array of machines to be authorized
     */
    onAuthorizeSelectedMachinesClicked(machines: Machine[]): void {
        this.authorizeSelectedMachines.emit(machines)
    }

    /**
     * Component constructor.
     * @param route ActivatedRoute used to get params from provided URL.
     * @param servicesApi Services API used to fetch machines from backend.
     * @param messageService Message service used to display feedback messages in UI.
     * @param location Location service used to update queryParams.
     */
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

    /**
     * Lazily loads machines table data.
     * @param event Event object containing an index of the first row, maximum
     * number of rows to be returned and machines filters.
     */
    loadData(event: TableLazyLoadEvent): void {
        // Indicate that machines refresh is in progress.
        this.dataLoading = true

        const authorized = this.prefilterValue ?? this.getTableFilterValue('authorized', event.filters)

        lastValueFrom(
            this.servicesApi.getMachines(
                event.first,
                event.rows,
                this.getTableFilterValue('text', event.filters),
                null,
                authorized
            )
        )
            .then((data) => {
                this.dataCollection = data.items ?? []
                this.totalRecords = data.total ?? 0
                this._unauthorizedInDataCollectionCount = this.dataCollection?.filter((m) => !m.authorized).length ?? 0
                if (authorized === false && this.hasFilter(this.table) === false) {
                    this.unauthorizedMachinesCount = this.totalRecords
                    this.unauthorizedMachinesCountChange.emit(this.totalRecords)
                } else {
                    this.fetchUnauthorizedMachinesCount()
                }

                if (this.selectedMachines.length > 0) {
                    // Clear selection on any lazy data load.
                    // This is to prevent confusion when selected machines could be out of filtered results.
                    this.clearSelection()
                }
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

    /**
     * Returns true if the table's data collection contains any authorized machine; false otherwise.
     */
    authorizedMachinesDisplayed(): boolean {
        return this.dataCollection?.some((m) => m.authorized) || false
    }

    /**
     * Returns true if the table's data collection contains any unauthorized machine; false otherwise.
     */
    unauthorizedMachinesDisplayed(): boolean {
        return this.dataCollection?.some((m) => !m.authorized) || false
    }

    /**
     * Fetches Unauthorized Machines Count via getUnauthorizedMachinesCount API.
     * @private
     */
    private fetchUnauthorizedMachinesCount(): void {
        lastValueFrom(this.servicesApi.getUnauthorizedMachinesCount())
            .then((count) => {
                this.unauthorizedMachinesCount = count ?? 0
                this.unauthorizedMachinesCountChange.emit(this.unauthorizedMachinesCount)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get Unauthorized Machines count',
                    detail: 'Error getting Unauthorized Machines count: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Deletes given machine from the table's data collection.
     * @param machineId id of the machine to be deleted
     */
    deleteMachine(machineId: number) {
        const idx = (this.dataCollection?.map((m) => m.id) || []).indexOf(machineId)
        if (idx >= 0) {
            this.dataCollection.splice(idx, 1)
            this.fetchUnauthorizedMachinesCount()
        }
    }

    /**
     * Refreshes given machine in the table's data collection.
     * @param machine machine to be refreshed
     */
    refreshMachineState(machine: Machine) {
        const idx = (this.dataCollection?.map((m) => m.id) || []).indexOf(machine.id)
        if (idx >= 0) {
            this.dataCollection.splice(idx, 1, machine)
        }
    }

    /**
     * Clears the machines selection.
     */
    clearSelection() {
        this.selectedMachines = []
        this.selectAll = false
    }

    /**
     * Setter for dataLoading property allowing parent component to enable/disable loading state.
     * @param loading value to be set
     */
    setDataLoading(loading: boolean) {
        this.dataLoading = loading
    }

    /**
     * Callback called when any single machine is selected or deselected.
     * @param selection current selection state
     */
    onSelectionChange(selection: Machine[]) {
        this.selectAll = selection.length > 0 && selection.length === this._unauthorizedInDataCollectionCount
    }

    /**
     * Callback called after click event on the Select all checkbox.
     * @param event change event containing 'checked' boolean flag and the original Click event
     */
    onSelectAllChange(event: TableSelectAllChangeEvent) {
        if (event.checked) {
            this.selectAll = true
            // Custom select all behavior: select only unauthorized machines visible on current table page.
            this.selectedMachines = this.dataCollection.filter((m) => !m.authorized)
            return
        }

        this.clearSelection()
    }
}
