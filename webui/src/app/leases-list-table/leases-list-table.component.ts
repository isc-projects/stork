import { Component, OnDestroy, OnInit, ViewChild, inject } from '@angular/core'
import { tableHasFilter, tableFiltersToQueryParams, convertSortingFields } from '../table'
import { DHCPService, Lease, LeaseListSortField, Leases } from '../backend'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { MenuItem, MessageService, PrimeTemplate, TableState, FilterMetadata } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { ManagedAccessDirective } from '../managed-access.directive'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { Button } from 'primeng/button'
import { FormsModule } from '@angular/forms'
import { FloatLabel } from 'primeng/floatlabel'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputNumber } from 'primeng/inputnumber'
import { InputText } from 'primeng/inputtext'
import { Tag } from 'primeng/tag'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { IdentifierComponent } from '../identifier/identifier.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { Tooltip } from 'primeng/tooltip'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { SplitButton } from 'primeng/splitbutton'
import { DaemonFilterComponent } from '../daemon-filter/daemon-filter.component'
import { stateToString } from '../lease-utils'

/**
 * This component implements a table of leases.
 * The list of leases is paged and can be filtered by provided
 * URL queryParams or by using form inputs responsible for
 * filtering.
 */
@Component({
    selector: 'app-leases-list-table',
    templateUrl: './leases-list-table.component.html',
    styleUrls: ['./leases-list-table.component.sass'],
    imports: [
        ManagedAccessDirective,
        ConfirmDialog,
        Button,
        RouterLink,
        TableModule,
        Tag,
        PrimeTemplate,
        FloatLabel,
        InputNumber,
        FormsModule,
        TriStateCheckboxComponent,
        IconField,
        InputIcon,
        InputText,
        IdentifierComponent,
        EntityLinkComponent,
        Tooltip,
        PluralizePipe,
        TableCaptionComponent,
        SplitButton,
        DaemonFilterComponent,
    ],
})
export class LeasesListTableComponent implements OnInit, OnDestroy {
    private router = inject(Router)
    private dhcpApi = inject(DHCPService)
    private messageService = inject(MessageService)

    /**
     * PrimeNG table instance.
     */
    @ViewChild('leasesListTable') table: Table

    /**
     * Flag stating whether table data is loading or not.
     */
    dataLoading: boolean

    /**
     * Total number of records displayed currently in the table.
     */
    totalRecords: number = 0

    /**
     * Data collection displayed currently in the table.
     */
    dataCollection: Lease[] = []

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

    /**
     * Loads leases from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for lease filtering. If it is
     * not specified, the current values are used when available.
     */
    loadData(event: TableLazyLoadEvent) {
        // Indicate that leases refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getLeaseList(JSON.stringify(event))
        lastValueFrom(
            this.dhcpApi.getLeaseList(
                event.first,
                event.rows,
                (event.filters['machineId'] as FilterMetadata)?.value ?? null,
                (event.filters['daemonId'] as FilterMetadata)?.value ?? null,
                (event.filters['subnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['localSubnetId'] as FilterMetadata)?.value ?? null,
                (event.filters['text'] as FilterMetadata)?.value || null,
                ...convertSortingFields<LeaseListSortField>(event)
            )
        )
            .then((data: Leases) => {
                this.dataCollection = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((err: any) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get leases list',
                    detail: 'Error getting leases list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    /**
     * Menu items of the splitButton which appears only for narrower viewports in the filtering toolbar.
     */
    toolbarButtons: MenuItem[] = []

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
                    map((f: { value: string }) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                    debounceTime(300),
                    distinctUntilChanged()
                )
                .subscribe((f: { filterConstraint: { value: any }; value: any }) => {
                    // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                    // so it's value must be set according to UI columnFilter value.
                    f.filterConstraint.value = f.value
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
                })
        )
    }

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.table?.clearFilterValues()
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
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
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
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'leases-list-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.table?.getStorage()
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
    protected readonly LeaseListSortField = LeaseListSortField

    /**
     * Callback called when autocomplete form emits any error message.
     * @param message error message
     */
    onAutocompleteError(message: string) {
        this.messageService.add({
            severity: 'error',
            summary: 'Cannot get daemons directory',
            detail: message,
            life: 10000,
        })
    }

    /**
     * Duplicate this function in the component so that the template can use it.
     */
    stateToString = stateToString
}
