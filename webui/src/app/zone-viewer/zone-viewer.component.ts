import { Component, DestroyRef, EventEmitter, Input, OnInit, Output, ViewChild } from '@angular/core'
import { ZoneRR } from '../backend/model/zoneRR'
import { debounceTime, distinctUntilChanged, lastValueFrom, map, Observable, Subject, tap } from 'rxjs'
import { getErrorMessage } from '../utils'
import { DNSRRType, DNSService, ZoneRRs } from '../backend'
import { FilterMetadata, LazyLoadEvent, MessageService } from 'primeng/api'
import { Table } from 'primeng/table'
import { tableHasFilter } from '../table'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'

/**
 * Interface describing the columns of the table.
 */
interface Column {
    name: string
    label: string
}

/**
 * Component fetching and displaying zone contents (resource records) in a table.
 *
 * It compacts presented data by removing the zone name (gathered from the SOA record)
 * from the resource records. It also omits the name from the resource record when
 * the previous resource record has the same name.
 */
@Component({
    selector: 'app-zone-viewer',
    standalone: false,
    templateUrl: './zone-viewer.component.html',
    styleUrl: './zone-viewer.component.sass',
})
export class ZoneViewerComponent implements OnInit {
    /**
     * Provides direct access to the the PrimeNG table component.
     */
    @ViewChild('table') table: Table

    /**
     * Holds the daemon ID.
     */
    @Input({ required: true }) daemonId: number

    /**
     * Holds the DNS view name.
     */
    @Input({ required: true }) viewName: string

    /**
     * Holds the zone ID.
     */
    @Input({ required: true }) zoneId: number

    /**
     * Emits the event indicating that fetching the zone resource records
     * has failed.
     *
     * The parent component can use this event to take specific actions
     * like hiding the zone viewer dialog.
     */
    @Output() viewerError = new EventEmitter<void>()

    /**
     * Holds the names of the columns to display in the table.
     */
    cols: Column[] = [
        { name: 'name', label: 'Name' },
        { name: 'ttl', label: 'TTL' },
        { name: 'rrClass', label: 'Class' },
        { name: 'rrType', label: 'Type' },
        { name: 'data', label: 'Data' },
    ]

    /**
     * RR types that can be selected to filter the zone contents.
     */
    rrTypes: string[] = []

    /**
     * Holds zone resource records fetched from the server.
     */
    zoneData: ZoneRR[] = []

    /**
     * Holds the timestamp of the last zone transfer.
     */
    zoneTransferAt: string = null

    /**
     * Holds the total number of zone resource records.
     */
    totalRecords: number = 0

    /**
     * Holds the flag indicating that the zone resource records are being loaded.
     */
    loading: boolean = false

    /**
     * Holds the default number of rows to display in the table.
     */
    rows: number = 10

    /**
     * Holds the name of the zone gathered from the SOA record.
     *
     * It is used in the _transformZoneRR function.
     */
    private _zoneName: string | null = null

    /**
     * Holds the name of the last processed resource record.
     *
     * It is used to omit repeated names in subsequent resource records.
     * It is used in the _transformZoneRR function.
     */
    private _lastName: string | null = null

    /**
     * RxJS Subject used for filtering RRs based on UI filtering form inputs.
     * @private
     */
    private _rrsTableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * Constructor.
     *
     * @param _dnsApi DNS API service.
     * @param _messageService message service.
     */
    constructor(
        private _dnsApi: DNSService,
        private _messageService: MessageService,
        private destroyRef: DestroyRef
    ) {}

    /**
     * Component lifecycle hook which inits the component.
     *
     * It initializes the RR types that can be selected to filter the zone contents.
     */
    ngOnInit(): void {
        this.rrTypes = Object.values(DNSRRType).map((type) => type.toString())
        this._rrsTableFilter$
            .pipe(
                map((f) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                debounceTime(300),
                distinctUntilChanged(),
                takeUntilDestroyed(this.destroyRef)
            )
            .subscribe((f) => {
                // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                // so it's value must be set according to UI columnFilter value.
                f.filterConstraint.value = f.value
                this.table.first = 0
                this.loadRRs(this.table.createLazyLoadMetadata())
            })
    }

    /**
     * Fetches the zone resource records from the server.
     *
     * It is called internally by loadRRs() and refreshRRsFromDNS() functions.
     *
     * @param req$ observable of the request.
     * @param refresh flag indicating if the request is a refresh.
     * @private
     */
    private _fetchRRs(req$: Observable<ZoneRRs>, refresh: boolean): void {
        this.loading = true
        lastValueFrom(
            req$.pipe(
                tap(() => this._beginTransformZoneRR()),
                map((data) => {
                    return {
                        items: data.items?.map((rr) => this._transformZoneRR(rr)) ?? [],
                        zoneTransferAt: data.zoneTransferAt,
                        total: data.total,
                    }
                })
            )
        )
            .then((data) => {
                // The data have been successfully loaded.
                this.zoneData = data.items ?? []
                this.zoneTransferAt = data.zoneTransferAt
                this.totalRecords = data.total
            })
            .catch((error) => {
                // Show the error message.
                const errorMsg = getErrorMessage(error)
                this._messageService.add({
                    severity: 'error',
                    summary: refresh ? 'Error refreshing zone contents from DNS server' : 'Error getting zone contents',
                    detail: errorMsg,
                    life: 10000,
                })
                // Notify the parent.
                this.viewerError.emit()
            })
            .finally(() => {
                this.loading = false
            })
    }

    /**
     * Loads the zone resource records from the server.
     *
     * @param event lazy load event containing pagination information.
     */
    loadRRs(event?: LazyLoadEvent): void {
        const req$ = this._dnsApi.getZoneRRs(
            this.daemonId,
            this.viewName,
            this.zoneId,
            event?.first ?? 0,
            event?.rows ?? this.rows,
            (event?.filters?.rrType as FilterMetadata)?.value ?? null,
            (event?.filters?.text as FilterMetadata)?.value ?? null
        )
        this._fetchRRs(req$, false)
    }

    /**
     * Refreshes the zone contents from the DNS server.
     *
     * @param event lazy load event containing pagination information.
     * @private
     */
    private _refreshRRs(event?: LazyLoadEvent): void {
        const req$ = this._dnsApi.putZoneRRsCache(
            this.daemonId,
            this.viewName,
            this.zoneId,
            event?.first ?? 0,
            event?.rows ?? this.rows,
            (event?.filters?.rrType as FilterMetadata)?.value ?? null,
            (event?.filters?.text as FilterMetadata)?.value ?? null
        )
        this._fetchRRs(req$, true)
    }

    /**
     * Refreshes the zone contents from the DNS server.
     */
    refreshRRsFromDNS() {
        this.table.first = 0
        this._refreshRRs(this.table?.createLazyLoadMetadata())
    }

    /**
     * Resets the zone name and the last name.
     *
     * This function must be called before transforming the resource records
     * using the _transformZoneRR function.
     * @private
     */
    private _beginTransformZoneRR(): void {
        this._zoneName = null
        this._lastName = null
    }

    /**
     * Transforms the resource record to an abbreviated form.
     *
     * The zone transfer returns a set of full resource records (i.e., they include
     * fully qualified names, the names are included for all zone records etc.).
     * The abbreviated form is often used in the zone files and the purpose of
     * transforming the resource records is to display them in the abbreviated form.
     *
     * The transformation to the abbreviated form is done in the following way:
     * - Use 'at' character as name for the SOA record.
     * - Remove the zone name from the name of the non-SOA record (leave partial name
     *   instead of the fully qualified name).
     * - Omit the name of the resource record if the previous resource record has the
     *   same name.
     *
     * @param rr resource record to transform.
     * @returns transformed resource record.
     * @private
     */
    private _transformZoneRR(rr: ZoneRR): ZoneRR {
        let name: string = ''
        switch (true) {
            case rr.rrType === 'SOA':
                // The 'at' symbol is used in the zone file to represent the zone name.
                name = '@'
                this._zoneName = rr.name
                break
            case this._lastName === rr.name:
                // If subsequent resource record has the same name as the previous one,
                // let's omit the name (leave it empty).
                break
            case !!this._zoneName:
                // If we have the zone name (which we should after processing the SOA record),
                // we can remove it from the current resource record name.
                name = rr.name.replace(`.${this._zoneName}`, '')
                break
            default:
                // In all other cases let's just use what we have.
                name = rr.name
        }
        // Remember the current name as the last processed name.
        this._lastName = rr.name

        // Return the transformed resource record.
        return {
            ...rr,
            name,
        }
    }

    /**
     * Reference to hasFilter() utility function so it can be used in the html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Filters the zone resource records table by RR type or text.
     *
     * @param value value of the filter.
     * @param filterConstraint filter constraint.
     */
    filterRRsTable(value: any, filterConstraint: FilterMetadata): void {
        this._rrsTableFilter$.next({ value, filterConstraint })
    }

    /**
     * Clears a value for given zone table filter constraint and reloads the table with the new filtering.
     * @param filterConstraint
     */
    clearFilter(filterConstraint: FilterMetadata) {
        filterConstraint.value = null
        this.table.first = 0
        this.loadRRs(this.table.createLazyLoadMetadata())
    }

    /**
     * Clears the PrimeNG table state (filtering, pagination are reset).
     */
    clearTableState() {
        this.table?.clear()
    }
}
