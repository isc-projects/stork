import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute, ParamMap } from '@angular/router'
import { Location } from '@angular/common'
import { debounceTime, Subject, Subscription } from 'rxjs'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { TableState } from 'primeng/api'
import { distinctUntilChanged, map } from 'rxjs/operators'

/**
 * Interface containing base properties that are supported when filtering table via URL queryParams.
 */
export interface BaseQueryParamFilter {
    text?: string
}

/**
 * Checks if given PrimeNG table filters contain any non-blank value.
 * @param filters PrimeNG table filters object
 * @param continueWhen callable that evaluates to boolean value; when evaluated to true, the filter for given filterKey is considered blank even if it has meaningful value
 * @return true if any non-blank filter was found; false otherwise
 */
export function hasFilter(
    filters: { [p: string]: FilterMetadata | FilterMetadata[] } = {},
    continueWhen: (filterKey: string) => boolean = () => false
): boolean {
    for (const [filterKey, filterMetadata] of Object.entries(filters)) {
        if (continueWhen(filterKey)) {
            continue
        }

        if (Array.isArray(filterMetadata)) {
            for (const filter of filterMetadata) {
                if (
                    (filter.matchMode != 'contains' && filter.value !== null) ||
                    (filter.matchMode == 'contains' && filter.value)
                ) {
                    return true
                }
            }
        } else if (filterMetadata) {
            if (
                (filterMetadata.matchMode != 'contains' && filterMetadata.value !== null) ||
                (filterMetadata.matchMode == 'contains' && filterMetadata.value)
            ) {
                return true
            }
        }
    }

    return false
}

/**
 * Parses string into boolean value. Returns boolean or null if it couldn't be parsed.
 * @param val
 */
export function parseBoolean(val: string): boolean | null {
    return val === 'true' ? true : val === 'false' ? false : null
}

/**
 * Abstract class unifying all components using PrimeNG table with data lazyLoading.
 * The class takes one generic argument, which is the type of the single record object
 * to be displayed in the table.
 * Derived class must implement abstract members:
 * - table field
 * - loadData(event) method
 */
export abstract class LazyLoadTable<TRecord> {
    /**
     * Holds the total amount of all records.
     * This field is supposed to be bound to PrimeNG Table "totalRecords" input property.
     * e.g. [totalRecords]="totalRecords".
     * It should be updated whenever table's data is lazily loaded from the backend.
     */
    totalRecords: number = 0

    /**
     * Indicates if the data is being fetched from the server.
     * This field is supposed to be bound to PrimeNG Table "loading" input property.
     * e.g. [loading]="dataLoading".
     * It should be updated whenever we want to indicate that data loading is in progress.
     */
    dataLoading: boolean = false

    /**
     * Array collection of objects to display in the table.
     * This field is supposed to be bound to PrimeNG Table "value" input property.
     * e.g. [value]="dataCollection".
     * It should be updated with fetched data whenever table's data is lazily loaded from the backend.
     * If no records were fetched from the backend, it should be an empty array.
     */
    dataCollection: TRecord[]

    /**
     * Template of the current page report element.
     * This field is supposed to be bound to PrimeNG Table "currentPageReportTemplate" input property.
     * e.g. [currentPageReportTemplate]="currentPageReportTemplate".
     */
    currentPageReportTemplate: string = '{currentPage} of {totalPages} pages'

    /**
     * PrimeNG table instance.
     * This field is supposed to be implemented in derived class.
     * Angular ViewChild decorator can be used to achieve that, e.g.
     * @ViewChild('someTable') table: Table
     */
    abstract table: Table

    /**
     * Callback to invoke when paging, sorting or filtering happens in lazy mode.
     * This method is supposed to be implemented in derived class and bound to
     * PrimeNG Table "onLazyLoad" output property EventEmitter.
     * e.g. (onLazyLoad)="loadData($event)"
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
 * The class takes two generic arguments:
 * - filter interface defining all supported filter fields
 * - the type of the single record object to be displayed in the table.
 *
 * Derived class must implement abstract members:
 * - fields where supported filters are configured
 * - field "table" which is abstract in base LazyLoadTable abstract class
 * - "loadData(event)" method which is abstract in base LazyLoadTable abstract class
 *
 * Derived class must also call onInit() in ngOnInit() and onDestroy() in ngOnDestroy().
 */
export abstract class PrefilteredTable<
    FilterInterface extends BaseQueryParamFilter,
    TRecord,
> extends LazyLoadTable<TRecord> {
    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription = new Subscription()

    /**
     * The provided queryParam filter RxJS Subject.
     */
    filter$ = new Subject<FilterInterface>()

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * The recent queryParam filter applied to the table data. Only filters that pass the
     * validation are used.
     */
    validFilter: FilterInterface = {} as FilterInterface

    /**
     * An array of errors found during queryParam filter validation.
     *
     * This array holds errors that were found during queryParam filter validation.
     * The intention is to use it to display feedback about errors in UI.
     */
    filterTextFormatErrors: string[] = []

    /**
     * queryParam keyword of the prefilter.
     */
    abstract prefilterKey: keyof FilterInterface

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey by appending either '-all' suffix or
     * numeric value, e.g. '-1', or boolean value as string, e.g. '-true'.
     *
     * Example:
     * stateKeyPrefix = 'hosts-table'
     * stateKey = 'hosts-table-all'
     * or
     * stateKey = 'hosts-table-3'
     */
    abstract stateKeyPrefix: string

    /**
     * Unique identifier of a stateful table used to store table's state in browser's storage.
     * This field is supposed to be bound to PrimeNG Table "stateKey" input property.
     * e.g. [stateKey]="stateKey".
     */
    stateKey: string

    /**
     * Keeps value of the pre-filter from queryParam (e.g. kea app Id).
     * If no pre-filtering is used, it will be null.
     */
    prefilterValue: number | boolean

    /**
     * Table's index of the first row to be displayed, restored from browser's storage.
     * @private
     */
    private _restoredFirst: number = 0

    /**
     * Table's number of rows to display per page, restored from browser's storage.
     * @private
     */
    private _restoredRows: number = 10

    /**
     * Keeps restored PrimeNG table. PrimeNG restores table's state from browser storage in ngOnChanges lifecycle hook of the table component,
     * that's why it can be accessed even before ngOnInit lifecycle hook. Restored table may be used to create LazyLoadMetadata when PrimeNG
     * table is not yet defined.
     * @private
     */
    private _restoredTable: Table

    /**
     * Array of all numeric keys of the FilterInterface. This should be a superset of queryParamNumericKeys.
     */
    abstract filterNumericKeys: (keyof FilterInterface)[]

    /**
     * Array of all boolean keys of the FilterInterface. This should be a superset of queryParamBooleanKeys.
     */
    abstract filterBooleanKeys: (keyof FilterInterface)[]

    /**
     * Array of all numeric keys of the FilterInterface that are supported when filtering via URL queryParams.
     * Note that it doesn't have to contain prefilterKey. prefilterKey by default is considered as a primary
     * queryParam filter key. This should be a subset of filterNumericKeys.
     */
    abstract queryParamNumericKeys: (keyof FilterInterface)[]

    /**
     * Array of all boolean keys of the FilterInterface that are supported when filtering via URL queryParams.
     * This should be a subset of filterBooleanKeys.
     */
    abstract queryParamBooleanKeys: (keyof FilterInterface)[]

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values, e.g. dhcpVersion=4|6.
     * A single FilterValidator contains filter key name and an array of allowed values for the filter.
     * E.g., {filterKey: 'dhcpVersion', allowedValues: [4, 6]}
     */
    abstract filterValidators: { filterKey: string; allowedValues: string[] | number[] }[]

    /**
     * Constructor of PrefilteredTable class. It requires ActivatedRoute and Location service to be passed by derived
     * class.
     * @param _route ActivatedRoute used to get params from provided URL.
     * @param _location Location service used to update queryParams.
     * @protected
     */
    protected constructor(
        private _route: ActivatedRoute,
        private _location: Location
    ) {
        super()
    }

    /**
     * Callback method called when PrimeNG table's state was saved to browser's storage.
     * This method is supposed to be bound to PrimeNG Table "onStateSave" output property EventEmitter.
     * e.g. (onStateSave)="stateSaved(table)"
     * @param table table which state was saved
     */
    stateSaved(table: Table): void {
        // Do not store selection as part of the table state.
        const state = JSON.parse(sessionStorage.getItem(this.stateKey))
        state.selection = []
        sessionStorage.setItem(this.stateKey, JSON.stringify(state))

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
     * This method is supposed to be bound to PrimeNG Table "onStateRestore" output property EventEmitter.
     * e.g. (onStateRestore)="stateRestored($event, table)"
     * @param state restored state
     * @param table table which state was restored
     */
    stateRestored(state: TableState, table: Table): void {
        if (table.restoringFilter) {
            // Force set this flag to false.
            // This is a workaround of the issue in PrimeNG,
            // where for stateful table, sometimes when filtering is applied,
            // table.first property is not set to 0 as expected.
            table.restoringFilter = false
        }

        // Backup restored data to properties.
        // They will be used when PrimeNG table is not available.
        // Use case: navigation back from detailed host view tab (index > 0)
        // to hosts' list view tab (index 0).
        this._restoredFirst = state.first
        this._restoredRows = state.rows
        this._restoredTable = table
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
    getTableFilterValue(k: string, filters?: { [p: string]: FilterMetadata | FilterMetadata[] }): any {
        if (filters) {
            return filters?.hasOwnProperty(k) ? (filters[k][0]?.value ?? (filters[k] as FilterMetadata).value) : null
        }

        if (!this.table?.filters?.hasOwnProperty(k)) {
            return null
        }

        return this.table.filters[k][0]?.value ?? (this.table.filters[k] as FilterMetadata).value
    }

    /**
     * Clean-up which should be done at ngOnDestroy() of derived class.
     */
    onDestroy(): void {
        this.filter$.complete()
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Initialization method which should be called at ngOnInit() of derived class.
     *
     * It extracts prefilterKey from ActivatedRoute snapshot queryParams, if it was provided.
     * Filter handlers are subscribed.
     */
    onInit(): void {
        this.dataLoading = true

        const paramMap = this._route.snapshot.paramMap
        const queryParamMap = this._route.snapshot.queryParamMap

        // Get param id and queryParam value for prefilterKey Id.
        const id = paramMap.get('id')
        if (!id || id === 'all') {
            this.parsePrefilterValueFromQueryParam(queryParamMap)
            this.stateKey = this.hasPrefilter()
                ? `${this.stateKeyPrefix}-${this.prefilterValue}`
                : `${this.stateKeyPrefix}-all`
        }

        this.subscribeFilterHandlers()
    }

    /**
     * Callback method called when PrimeNG table was filtered.
     * This method is supposed to be bound to PrimeNG Table "onFilter" output property EventEmitter.
     * e.g. (onFilter)="onFilter()"
     */
    onFilter(): void {
        let change = false
        for (const k of Object.keys(this.validFilter)) {
            if (this.validFilter[k] != null && this.getTableFilterValue(k) != this.validFilter[k]) {
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
     * Returns true if the prefilter from queryParam was applied; false otherwise.
     */
    hasPrefilter(): boolean {
        return this.prefilterValue != null
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

        // Even when all filters are cleared, restore the prefilter value if it was given in queryParams.
        // Note that other queryParam filters are also cleared here.
        if (this.hasPrefilter()) {
            table.filters[this.prefilterKey as string] = { value: this.prefilterValue, matchMode: 'equals' }
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
        return hasFilter(table?.filters, (filterKey) => this.hasPrefilter() && filterKey == this.prefilterKey)
    }

    /**
     * Updates the filter structure using URL query parameters.
     *
     * This update is triggered when the URL changes.
     * @param params query parameters received from activated route.
     */
    updateFilterFromQueryParameters(params: ParamMap): void {
        const numericKeys = this.queryParamNumericKeys.includes(this.prefilterKey)
            ? this.queryParamNumericKeys
            : [this.isPrefilterNumber() ? (this.prefilterKey ?? null) : null, ...this.queryParamNumericKeys]

        const filter: BaseQueryParamFilter = {}
        filter.text = params.get('text')

        for (const key of numericKeys) {
            // Convert the value to a number. It is NaN if the parameter
            // doesn't exist or it is malformed.
            if (params.has(key as string)) {
                const value = parseInt(params.get(key as string))
                filter[key as any] = isNaN(value) ? null : value
            }
        }

        const booleanKeys = this.queryParamBooleanKeys.includes(this.prefilterKey)
            ? this.queryParamBooleanKeys
            : [this.isPrefilterBoolean() ? (this.prefilterKey ?? null) : null, ...this.queryParamBooleanKeys]

        for (const key of booleanKeys) {
            if (params.has(key as string)) {
                filter[key as any] = parseBoolean(params.get(key as string))
            }
        }

        this.filter$.next(filter as FilterInterface)
    }

    /**
     * Triggers data load in the table without any filtering applied.
     */
    loadDataWithoutFilter(): void {
        this.filter$.next({} as FilterInterface)
    }

    /**
     * Triggers data load in the table with the valid filter applied.
     */
    loadDataWithValidFilter(): void {
        this.filter$.next(this.validFilter)
    }

    /**
     * Emits next value and filterConstraint for the table's filter,
     * which in the end will result in applying the filter on the table's data.
     * @param value value of the filter
     * @param filterConstraint filter field which will be filtered
     */
    filterTable(value: any, filterConstraint: FilterMetadata): void {
        this._tableFilter$.next({ value, filterConstraint })
    }

    /**
     * Subscribes handlers to the filter$ and _tableFilter$ observables.
     * @private
     */
    private subscribeFilterHandlers(): void {
        // Update the filter representation when the filtering parameters change.
        this._subscriptions.add(
            this.filter$.subscribe((filter) => {
                this.filterTextFormatErrors = this.validateFilter(filter)
                if (this.filterTextFormatErrors.length === 0) {
                    this.validFilter = filter
                }

                if (this.table) {
                    if (this.validatedFilterAndTableFilterDiffer()) {
                        // queryParams vs restored filter differs, overwrite.
                        this.table.first = 0
                        this.table.firstChange.emit(this.table.first)
                        this.table.filters = this.createTableFilter()

                        this.table.saveState()
                    }

                    this.reloadData(this.table)
                } else if (this._restoredTable) {
                    // PrimeNG table undefined but restoredTable defined, call onLazyLoad() using restored state.
                    this.loadData(this._restoredTable.createLazyLoadMetadata())
                } else {
                    // both PrimeNG table and restoredTable undefined, calling onLazyLoad() with constructed lazyLoadEvent.
                    const filters = this.createTableFilter()
                    this.loadData({
                        first: this._restoredFirst,
                        rows: this._restoredRows,
                        filters: filters,
                    })
                }
            })
        )

        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        this.table._filter()
                    })
                )
                .subscribe()
        )
    }

    /**
     * Update the URL query parameters basing on current validFilter.
     *
     * This function uses the Location provider instead Router or
     * ActivatedRoute to avoid re-rendering the component.
     * @private
     */
    private updateQueryParameters() {
        const params = []

        for (const key of Object.keys(this.validFilter)) {
            if (this.validFilter[key] != null) {
                params.push(`${encodeURIComponent(key)}=${encodeURIComponent(this.validFilter[key])}`)
            }
        }

        const baseUrl = this._route.snapshot.url.join('/')
        this._location.go(`/${baseUrl}`, params.join('&'))
    }

    /**
     * Checks if the provided filter is valid.
     * @param filter A filter to validate
     * @returns List of validation issues. If the list is empty, the filter is
     * valid.
     * @private
     */
    private validateFilter(filter: FilterInterface): string[] {
        const errors: string[] = []

        for (const key of this.filterNumericKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(`Please specify ${String(key)} as a number (e.g., ${String(key)}=4).`)
            }
        }

        for (const key of this.filterBooleanKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(
                    `Please specify ${String(key)} as a boolean (e.g., ${String(key)}=true or ${String(key)}=false).`
                )
            }
        }

        for (const validator of this.filterValidators) {
            if (
                filter.hasOwnProperty(validator.filterKey) &&
                !(validator.allowedValues as any[]).includes(filter[validator.filterKey])
            ) {
                errors.push(`Filter ${validator.filterKey} allows only values: ${validator.allowedValues.join(', ')}.`)
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
            this.validFilter[this.prefilterKey] != this.getTableFilterValue(this.prefilterKey as string)
        ) {
            return true
        }

        // 'text' queryParam filter may always be there, so it is also always checked.
        if (this.validFilter.text && this.validFilter.text != this.getTableFilterValue('text')) {
            return true
        }

        // Now let's compare all filterNumericKeys filters.
        for (const k of this.filterNumericKeys) {
            if (this.validFilter.hasOwnProperty(k) && this.validFilter[k] != this.getTableFilterValue(k as string)) {
                return true
            }
        }

        // Now let's compare all filterBooleanKeys filters.
        for (const k of this.filterBooleanKeys) {
            if (this.validFilter.hasOwnProperty(k) && this.validFilter[k] != this.getTableFilterValue(k as string)) {
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

        filter['text'] = {
            value: this.validFilter.hasOwnProperty('text') ? this.validFilter.text : null,
            matchMode: 'contains',
        }

        for (const k of this.filterNumericKeys) {
            filter[k as string] = { value: this.validFilter[k] ?? null, matchMode: 'equals' }
        }

        for (const k of this.filterBooleanKeys) {
            filter[k as string] = {
                value: this.validFilter.hasOwnProperty(k) ? this.validFilter[k] : null,
                matchMode: 'equals',
            }
        }

        return filter
    }

    /**
     * Parses value for the queryParam prefilter keyword and stores this value under prefilterValue.
     * @param queryParamMap
     * @private
     */
    private parsePrefilterValueFromQueryParam(queryParamMap: ParamMap): void {
        const queryParamValue = queryParamMap.get(this.prefilterKey as string)
        if (this.isPrefilterNumber()) {
            const numericId = parseInt(queryParamValue)
            this.prefilterValue = isNaN(numericId) ? null : numericId
        } else if (this.isPrefilterBoolean()) {
            this.prefilterValue = parseBoolean(queryParamValue)
        }
    }

    /**
     * Returns true if the prefilter is a numeric filter; false otherwise.
     * @private
     */
    private isPrefilterNumber(): boolean {
        return this.filterNumericKeys.includes(this.prefilterKey)
    }

    /**
     * Returns true if the prefilter is a boolean filter; false otherwise.
     * @private
     */
    private isPrefilterBoolean(): boolean {
        return this.filterBooleanKeys.includes(this.prefilterKey)
    }
}
