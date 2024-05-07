import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ParamMap, Router } from '@angular/router'
import { Location } from '@angular/common'
import { Subject, Subscription } from 'rxjs'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { filter, map } from 'rxjs/operators'

/**
 * Interface containing base properties that are supported when filtering table via URL queryParams.
 */
export interface BaseQueryParamFilter {
    text?: string
}

/**
 * Abstract class unifying all components using PrimeNG table with data lazyLoading.
 */
export abstract class LazyLoadTable {
    /**
     * Holds the total amount of all records.
     */
    totalRecords: number = 0

    /**
     * Indicates if the data is being fetched from the server.
     */
    dataLoading: boolean = false

    /**
     * PrimeNG table instance.
     */
    abstract table: Table

    /**
     * Template of the current page report element.
     */
    currentPageReportTemplate: string = '{currentPage} of {totalPages} pages'

    /**
     * Callback to invoke when paging, sorting or filtering happens in lazy mode.
     * @param {TableLazyLoadEvent} event - lazy load event which holds information about pagination, sorting and filtering
     */
    abstract loadData(event: TableLazyLoadEvent): void

    /**
     * Reloads data in given table.
     * @param table table when data is to be reloaded
     */
    reloadData(table: Table): void {
        table.onLazyLoad.emit(table.createLazyLoadMetadata())
    }
}

/**
 * Abstract class unifying all components using PrimeNG table with data lazyLoading, that
 * additionally provide filtering possibility. It is possible to pre-filter table data using
 * URL queryParams. Filtering is stateful, meaning that it's state is saved to browser's session
 * storage and restored when needed. So is table's pagination and sorting.
 */
export abstract class PrefilteredTable<FilterInterface extends BaseQueryParamFilter> extends LazyLoadTable {
    /**
     * Router service used to update queryParams.
     * @private
     */
    private _router: Router

    /**
     * Location service used to update queryParams.
     * @private
     */
    private _location: Location

    /**
     * The provided filter.
     * The source property indicates where the filter comes from:
     * - input - the filter is set by the user in the input box,
     * - callback - the filter is set by the child component,
     * - query - the filter is set by the URL query parameters,
     * - state - the filter is restored from saved state.
     */
    filter$ = new Subject<{ source: 'input' | 'callback' | 'query' | 'state'; filter: FilterInterface }>()

    /**
     * The recent filter applied to the table data. Only filters that pass the
     * validation are used.
     */
    validFilter: FilterInterface = {} as FilterInterface

    /**
     * An array of errors in specifying filter text.
     *
     * This array holds errors displayed next to the host filtering input box.
     */
    filterTextFormatErrors: string[] = []

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     */
    abstract subscriptions: Subscription

    /**
     * queryParam keyword of the filter by Id.
     */
    abstract prefilterKey: keyof FilterInterface

    /**
     * Unique identifier of a stateful table used to store table's state in browser's storage.
     */
    abstract stateKey: string

    /**
     * Keeps table unique Id which comes from queryParam filtering by Id (e.g. kea app Id).
     * If no pre-filtering by Id is used, it will be null.
     */
    tableId: number

    /**
     * Table's index of the first row to be displayed, restored from browser's storage.
     */
    restoredFirst: number = 0

    /**
     * Table's number of rows to display per page, restored from browser's storage.
     */
    restoredRows: number = 10

    /**
     * Keeps restored PrimeNG table. PrimeNG restores table's state from browser storage in ngOnChanges lifecycle hook of the table component,
     * that's why it can be accessed even before ngOnInit lifecycle hook. Restored table may be used to create LazyLoadMetadata when hostsTable
     * is not yet defined.
     */
    restoredTable: Table

    /**
     * Array of all numeric keys of the FilterInterface.
     */
    abstract filterNumericKeys: (keyof FilterInterface)[]

    /**
     * Array of all boolean keys of the FilterInterface.
     */
    abstract filterBooleanKeys: (keyof FilterInterface)[]

    /**
     * Array of all numeric keys of the FilterInterface that are supported when filtering via URL queryParams.
     * Note that it doesn't have to contain prefilterKey. prefilterKey by default is considered as a primary
     * queryParam filter key.
     */
    abstract queryParamNumericKeys: (keyof FilterInterface)[]

    /**
     * Array of all boolean keys of the FilterInterface that are supported when filtering via URL queryParams.
     */
    abstract queryParamBooleanKeys: (keyof FilterInterface)[]

    /**
     * Constructor of PrefilteredTable class. It requires Router and Location services to be passed by derived
     * class.
     * @param router Router service used to update queryParams.
     * @param location Location service used to update queryParams.
     * @protected
     */
    protected constructor(router: Router, location: Location) {
        super()
        this._router = router
        this._location = location
    }

    /**
     * Parses value for the queryParam "by Id" keyword and stores this value under tableId.
     * @param queryParamMap
     */
    parseIdFromQueryParam(queryParamMap: ParamMap): void {
        const id = parseInt(queryParamMap.get(this.prefilterKey as string))
        this.tableId = isNaN(id) ? null : id
    }

    /**
     * Callback method called when PrimeNG table's state was saved to browser's storage.
     * @param table table which state was saved
     */
    stateSaved(table: Table): void {
        if (table.restoringFilter) {
            // Force set this flag to false.
            // This is a workaround of the issue in PrimeNG,
            // where for stateful table, sometimes when filtering is applied,
            // table.first property is not set to 0 as expected.
            table.restoringFilter = false
        }
    }

    /**
     * Callback method called when PrimeNG table's state was restored from browser's storage.
     * @param state restored state
     * @param hostsTable table which state was restored
     */
    stateRestored(state: any, hostsTable: Table): void {
        if (hostsTable.restoringFilter) {
            // Force set this flag to false.
            // This is a workaround of the issue in PrimeNG,
            // where for stateful table, sometimes when filtering is applied,
            // table.first property is not set to 0 as expected.
            hostsTable.restoringFilter = false
        }

        // Backup restored data to properties.
        // They will be used when hostTable is not available.
        // Use case: navigation back from detailed host view tab (index > 0)
        // to hosts' list view tab (index 0).
        this.restoredFirst = state.first
        this.restoredRows = state.rows
        this.restoredTable = hostsTable
    }

    /**
     * Returns table filter value for the given filter key-name.
     *
     * PrimeNG Table has a little confusing logic with keeping table's filters
     * as either FilterMetadata or an array of FilterMetadata:
     * table.filters: {[p: string]: FilterMetadata | FilterMetadata[]}
     *
     * This helper method checks if the filter value exists firstly in FilterMetadata[].
     * If not, then it checks if it exists as value in FilterMetadata.
     *
     * If no filter value is found, null is returned.
     *
     * @param k filter name key
     * @param filters filters object where the filter is checked. If undefined, this.table property filters are checked.
     */
    getTableFilterVal(k: string, filters?: { [p: string]: FilterMetadata | FilterMetadata[] }): any {
        if (!filters) {
            return this.table?.filters?.hasOwnProperty(k)
                ? this.table.filters[k][0]?.value ?? (this.table.filters[k] as FilterMetadata).value
                : null
        } else {
            return filters?.hasOwnProperty(k) ? filters[k][0]?.value ?? (filters[k] as FilterMetadata).value : null
        }
    }

    /**
     * Callback method called when PrimeNG table was filtered.
     */
    onFilter(): void {
        let change = false
        for (const k of Object.keys(this.validFilter)) {
            if (this.validFilter[k] != null && this.getTableFilterVal(k) != this.validFilter[k]) {
                // This filter was either cleared or edited, so delete it from validFilter.
                change = true
                delete this.validFilter[k]
            }
        }

        if (change) {
            this.updateQueryParameters()
        }
    }

    /**
     * Returns true if prefilter by Id from queryParam was applied; false otherwise.
     */
    hasPrefilter(): boolean {
        return this.tableId != null
    }

    /**
     * Clears filtering on the table and stores table's state.
     * @param table table where filters are to be cleared
     */
    clearFilters(table: Table): void {
        // Clear filters in table.
        table.clearFilterValues()

        // Clear queryParam filters parsing errors.
        this.filterTextFormatErrors = []

        // Even when all filters are cleared, restore "by Id" filter if it was given in queryParams.
        // Note that other queryParam filters are also cleared here.
        if (this.hasPrefilter()) {
            table.filters[this.prefilterKey as string] = { value: this.tableId, matchMode: 'equals' }
        }

        table.first = 0
        table.firstChange.emit(table.first)
        table.saveState()

        // Reload data with cleared filters.
        this.reloadData(table)

        table.onFilter.emit()
    }

    /**
     * Checks whether given table has any active filters applied.
     * @param table table which filters are checked
     */
    hasFilter(table: Table): boolean {
        if (table.filters) {
            for (const [filterKey, filterMetadata] of Object.entries(table.filters)) {
                if (this.hasPrefilter() && filterKey == this.prefilterKey) {
                    // If this is filtered view by Id from queryParams, don't count it as an active filter.
                    continue
                }

                if (Array.isArray(filterMetadata)) {
                    for (let filter of filterMetadata) {
                        if (filter.value) {
                            return true
                        }
                    }
                } else if (filterMetadata) {
                    if (filterMetadata.value) {
                        return true
                    }
                }
            }
        }

        return false
    }

    /**
     * Updates the filter structure using URL query parameters.
     *
     * This update is triggered when the URL changes.
     * @param params query parameters received from activated route.
     */
    updateFilterFromQueryParameters(params: ParamMap): void {
        const numericKeys =
            !this.prefilterKey || this.prefilterKey in this.queryParamNumericKeys
                ? this.queryParamNumericKeys
                : [this.prefilterKey, ...this.queryParamNumericKeys]

        let filter: BaseQueryParamFilter = {}
        filter.text = params.get('text')

        for (let key of numericKeys) {
            // Convert the value to a number. It is NaN if the parameter
            // doesn't exist or it is malformed.
            if (params.has(key as string)) {
                const value = parseInt(params.get(key as string))
                filter[key as any] = isNaN(value) ? null : value
            }
        }

        const parseBoolean = (val: string) => (val === 'true' ? true : val === 'false' ? false : null)

        for (let key of this.queryParamBooleanKeys) {
            if (params.has(key as string)) {
                filter[key as any] = parseBoolean(params.get(key as string))
            }
        }

        this.filter$.next({
            source: 'query',
            filter: filter as FilterInterface,
        })
    }

    /**
     * Pipes filter validation to the hostFilter$ subject.
     */
    subscribeFilterValidation(): void {
        // Pipe the valid filter to the hostFilter$ subject.
        this.subscriptions.add(
            this.filter$
                .pipe(
                    // Valid filter has no validation errors.
                    filter((f) => this.validateFilter(f.filter).length === 0),
                    map((f) => f.filter)
                )
                .subscribe((filter) => {
                    // Remember the filter.
                    this.validFilter = filter
                })
        )
    }

    /**
     * Subscribes handler to the filter$ observable.
     */
    subscribeFilterHandler(): void {
        // Update the filter representation when the filtering parameters change.
        this.subscriptions.add(
            this.filter$.subscribe((f) => {
                this.filterTextFormatErrors = this.validateFilter(f.filter)

                if (this.table) {
                    if (this.validatedFilterAndTableFilterDiffer()) {
                        // queryParams vs restored filter differs, overwrite.
                        this.table.first = 0
                        this.table.firstChange.emit(this.table.first)
                        this.table.filters = this.createTableFilter()

                        this.table.saveState()
                    }

                    this.reloadData(this.table)
                } else if (this.restoredTable) {
                    // hostTable undefined but restoredTable defined, call onLazyLoad() using restored state.
                    this.loadData(this.restoredTable.createLazyLoadMetadata())
                } else {
                    // both hostTable and restoredTable undefined, calling onLazyLoad() with constructed lazyLoadEvent.
                    const filters = this.createTableFilter()
                    this.loadData({
                        first: this.restoredFirst,
                        rows: this.restoredRows,
                        filters: filters,
                    })
                }
            })
        )
    }

    /**
     * Update the URL query parameters basing on current validFilter.
     *
     * This function uses the Location provider instead Router or
     * ActivatedRoute to avoid re-rendering the component.
     */
    private updateQueryParameters() {
        const params = []

        for (let key of Object.keys(this.validFilter)) {
            if (this.validFilter[key] != null) {
                params.push(`${encodeURIComponent(key)}=${encodeURIComponent(this.validFilter[key])}`)
            }
        }

        const baseUrl = this._router.url.split('?')[0]
        this._location.go(baseUrl, params.join('&'))
    }

    /**
     * Checks if the provided filter is valid.
     * @param filter A filter to validate
     * @returns List of validation issues. If the list is empty, the filter is
     * valid.
     */
    private validateFilter(filter: FilterInterface): string[] {
        const errors: string[] = []

        for (let key of this.filterNumericKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(`Please specify ${String(key)} as a number (e.g., ${String(key)}=2).`)
            }
        }

        for (let key of this.filterBooleanKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(
                    `Please specify ${String(key)} as a boolean (e.g., ${String(key)}=true or ${String(key)}=false).`
                )
            }
        }

        return errors
    }

    /**
     * Compares values of the validFilter and table's actual filter.
     * @private
     * @returns true if filters differ; false otherwise
     */
    private validatedFilterAndTableFilterDiffer(): boolean {
        // If prefilterKey is defined, it is always being checked.
        if (
            this.validFilter.hasOwnProperty(this.prefilterKey) &&
            this.validFilter[this.prefilterKey] != this.getTableFilterVal(this.prefilterKey as string)
        ) {
            return true
        }

        // 'text' queryParam filter may always be there, so it is also always checked.
        if (this.validFilter.text && this.validFilter.text != this.getTableFilterVal('text')) {
            return true
        }

        // Now let's compare all filterNumericKeys filters.
        for (let k of this.filterNumericKeys) {
            if (this.validFilter.hasOwnProperty(k) && this.validFilter[k] != this.getTableFilterVal(k as string)) {
                return true
            }
        }

        // Now let's compare all filterBooleanKeys filters.
        for (let k of this.filterBooleanKeys) {
            if (this.validFilter.hasOwnProperty(k) && this.validFilter[k] != this.getTableFilterVal(k as string)) {
                return true
            }
        }

        // No diff found.
        return false
    }

    /**
     * Creates FilterMetadata from actual validFilter and returns FilterMetadata object.
     * @private
     */
    private createTableFilter(): { [p: string]: FilterMetadata | FilterMetadata[] } {
        const filter: { [s: string]: FilterMetadata | FilterMetadata[] } = {}

        if (this.prefilterKey && this.validFilter.hasOwnProperty(this.prefilterKey)) {
            filter[this.prefilterKey as string] = { value: this.validFilter[this.prefilterKey], matchMode: 'equals' }
        } else if (this.prefilterKey) {
            filter[this.prefilterKey as string] = { value: null, matchMode: 'equals' }
        }

        if (this.validFilter.hasOwnProperty('text')) {
            filter['text'] = [{ value: this.validFilter.text, matchMode: 'contains' }]
        }

        for (let k of this.filterNumericKeys) {
            filter[k as string] = { value: this.validFilter[k] ?? null, matchMode: 'equals' }
        }

        for (let k of this.filterBooleanKeys) {
            filter[k as string] = {
                value: this.validFilter.hasOwnProperty(k) ? this.validFilter[k] : null,
                matchMode: 'equals',
            }
        }

        return filter
    }
}
