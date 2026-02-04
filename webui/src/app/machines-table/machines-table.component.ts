import { Component, effect, EventEmitter, Input, OnDestroy, OnInit, Output, signal, ViewChild } from '@angular/core'
import { convertSortingFields, tableFiltersToQueryParams, tableHasFilter } from '../table'
import { Machine, MachineSortField, ServicesService } from '../backend'
import { Table, TableLazyLoadEvent, TableModule, TableSelectAllChangeEvent } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { MenuItem, MessageService, PrimeTemplate, TableState } from 'primeng/api'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { Message } from 'primeng/message'
import { NgFor, NgIf } from '@angular/common'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { FormsModule } from '@angular/forms'
import { InputText } from 'primeng/inputtext'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { ProgressBar } from 'primeng/progressbar'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { DaemonStatusComponent } from '../daemon-status/daemon-status.component'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { SplitButton } from 'primeng/splitbutton'

/**
 * This component is dedicated to display the table of Machines. It supports
 * lazy data loading from the backend and machines filtering.
 */
@Component({
    selector: 'app-machines-table',
    templateUrl: './machines-table.component.html',
    styleUrl: './machines-table.component.sass',
    imports: [
        NgIf,
        Button,
        ManagedAccessDirective,
        TableModule,
        HelpTipComponent,
        PrimeTemplate,
        TriStateCheckboxComponent,
        IconField,
        InputIcon,
        FormsModule,
        InputText,
        RouterLink,
        VersionStatusComponent,
        NgFor,
        ProgressBar,
        Message,
        LocaltimePipe,
        PlaceholderPipe,
        PluralizePipe,
        DaemonStatusComponent,
        TableCaptionComponent,
        SplitButton,
    ],
})
export class MachinesTableComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG table instance.
     */
    @ViewChild('table') machinesTable: Table

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
    selectedMachines = signal<Machine[]>([])

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
     * Machines currently displayed in the table.
     */
    dataCollection: Machine[] = []

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
     * Flag keeping track of whether table data is loading.
     */
    dataLoading: boolean

    /**
     * Number of records currently displayed in the table.
     */
    totalRecords: number

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

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
     * Menu items of the splitButton which appears only for narrower viewports in the filtering toolbar.
     */
    toolbarButtons: MenuItem[] = []

    /**
     * This flag states whether user has privileges to authorize machines.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canAuthorizeMachine = signal<boolean>(false)

    /**
     * Effect signal reacting on user privileges changes and triggering update of the splitButton model
     * inside the filtering toolbar.
     */
    privilegesChangeEffect = effect(() => {
        if (this.canAuthorizeMachine() || this.unauthorizedMachinesDisplayed() || this.selectedMachines().length) {
            this._updateToolbarButtons()
        }
    })

    /**
     * Updates filtering toolbar splitButton menu items.
     * Based on user privileges some menu items may be disabled or not.
     * @private
     */
    private _updateToolbarButtons() {
        const buttons: MenuItem[] = [
            {
                label: 'Authorize selected',
                icon: 'pi pi-lock',
                command: () => this.onAuthorizeSelectedMachinesClicked(this.selectedMachines()),
                disabled:
                    !this.canAuthorizeMachine() ||
                    !this.unauthorizedMachinesDisplayed() ||
                    !this.selectedMachines().length,
            },
        ]
        this.toolbarButtons = [...buttons]
    }

    /**
     * Component constructor.
     * @param servicesApi Services API used to fetch machines from backend.
     * @param messageService Message service used to display feedback messages in UI.
     * @param router Angular router used to trigger navigations.
     */
    constructor(
        private servicesApi: ServicesService,
        private messageService: MessageService,
        private router: Router
    ) {}

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        this._restoreTableRowsPerPage()
        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    map((f) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                    debounceTime(300),
                    distinctUntilChanged()
                )
                .subscribe((f) => {
                    // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                    // so it's value must be set according to UI columnFilter value.
                    f.filterConstraint.value = f.value
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.machinesTable) })
                })
        )
        this._updateToolbarButtons()
    }

    /**
     * Keeps state whether authorized machines are shown.
     */
    authorizedShown = signal<boolean>(null)

    /**
     * Lazily loads machines table data.
     * @param event Event object containing an index of the first row, maximum
     * number of rows to be returned and machines filters.
     */
    loadData(event: TableLazyLoadEvent): void {
        // Indicate that machines refresh is in progress.
        this.dataLoading = true

        const authorized = (event.filters['authorized'] as FilterMetadata)?.value ?? null
        this.authorizedShown.set(authorized)

        lastValueFrom(
            this.servicesApi.getMachines(
                event.first,
                event.rows,
                (event.filters['text'] as FilterMetadata)?.value || null,
                authorized,
                ...convertSortingFields<MachineSortField>(event)
            )
        )
            .then((data) => {
                this.dataCollection = data.items ?? []
                this.authorizedMachinesDisplayed.set(this.dataCollection.some((m) => m.authorized) || false)
                this.unauthorizedMachinesDisplayed.set(this.dataCollection.some((m) => !m.authorized) || false)
                this.totalRecords = data.total ?? 0
                this._unauthorizedInDataCollectionCount = this.dataCollection?.filter((m) => !m.authorized).length ?? 0
                if (
                    authorized === false &&
                    this.tableHasFilter(this.machinesTable, (filterKey) => filterKey === 'authorized') === false
                ) {
                    this.unauthorizedMachinesCount = this.totalRecords
                    this.unauthorizedMachinesCountChange.emit(this.totalRecords)
                } else {
                    this.fetchUnauthorizedMachinesCount()
                }

                if (this.selectedMachines().length > 0) {
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
    authorizedMachinesDisplayed = signal<boolean>(false)

    /**
     * Returns true if the table's data collection contains any unauthorized machine; false otherwise.
     */
    unauthorizedMachinesDisplayed = signal<boolean>(false)

    /**
     * Fetches Unauthorized Machines Count via getUnauthorizedMachinesCount API.
     */
    fetchUnauthorizedMachinesCount(): void {
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
     * Clears the machines selection.
     */
    clearSelection() {
        this.selectedMachines.set([])
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
            this.selectedMachines.set(this.dataCollection.filter((m) => !m.authorized))
            return
        }

        this.clearSelection()
    }

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.machinesTable?.clearFilterValues()
        this.router.navigate([])
    }

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     *
     * @param value
     * @param filterConstraint
     * @param debounceMode
     */
    filterTable(value: any, filterConstraint: FilterMetadata, debounceMode = true): void {
        if (debounceMode) {
            this._tableFilter$.next({ value, filterConstraint })
            return
        }

        filterConstraint.value = value
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.machinesTable) })
    }

    /**
     * Reference to the function so it can be used in html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.machinesTable) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'machines-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.machinesTable?.getStorage()
        storage?.setItem(this._tableStateStorageKey, JSON.stringify(state))
    }

    /**
     * Restores only rows per page count for the table from the state stored in user browser storage.
     * @private
     */
    private _restoreTableRowsPerPage() {
        const stateString = localStorage.getItem(this._tableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.rows = state.rows ?? 10
        }
    }

    /**
     * Reference to an enum so it could be used in the HTML template.
     * @protected
     */
    protected readonly MachineSortField = MachineSortField
}
