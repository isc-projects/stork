import { ChangeDetectorRef, Component, DestroyRef, inject, NgZone, OnInit, ViewChild } from '@angular/core'
import { FilterMetadata, MenuItem, MessageService, TableState } from 'primeng/api'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import {
    Daemon,
    DNSService,
    MachinesDirectoryEntry,
    ServicesService,
    ZoneTransferSortField,
    ZoneTransferState,
} from '../backend'
import { debounceTime, distinctUntilChanged, lastValueFrom, map, Subject } from 'rxjs'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { Router } from '@angular/router'
import { convertSortingFields, tableFiltersToQueryParams, tableHasFilter } from '../table'
import { getErrorMessage } from '../utils'
import { UnrootPipe } from '../pipes/unroot.pipe'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { TooltipModule } from 'primeng/tooltip'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ButtonModule } from 'primeng/button'
import { FieldsetModule } from 'primeng/fieldset'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { FloatLabelModule } from 'primeng/floatlabel'
import { MultiSelectModule } from 'primeng/multiselect'
import { FormsModule } from '@angular/forms'
import { NgTemplateOutlet } from '@angular/common'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'
import { SplitButtonModule } from 'primeng/splitbutton'
import { InputTextModule } from 'primeng/inputtext'
import { SelectModule } from 'primeng/select'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { CheckboxModule } from 'primeng/checkbox'
import { TagModule } from 'primeng/tag'

/**
 * A component that displays a table of DNS zone transfers.
 */
@Component({
    selector: 'app-zone-transfers-page',
    templateUrl: './zone-transfers-page.component.html',
    styleUrl: './zone-transfers-page.component.sass',
    imports: [
        BreadcrumbsComponent,
        ButtonModule,
        CheckboxModule,
        DurationPipe,
        EntityLinkComponent,
        FieldsetModule,
        FloatLabelModule,
        FormsModule,
        HumanCountPipe,
        IconFieldModule,
        InputIconModule,
        InputTextModule,
        LocaltimePipe,
        MultiSelectModule,
        NgTemplateOutlet,
        PlaceholderPipe,
        SelectModule,
        SplitButtonModule,
        TableCaptionComponent,
        TableModule,
        TabViewComponent,
        TagModule,
        TooltipModule,
        UnrootPipe,
    ],
})
export class ZoneTransfersPageComponent implements OnInit {
    /**
     * Angular change detection required to manually trigger detectChanges in this component.
     * @private
     */
    private cd = inject(ChangeDetectorRef)

    /**
     * Service providing DNS REST APIs.
     * @private
     */
    private dnsService = inject(DNSService)

    /**
     * Service providing machines REST APIs.
     * @private
     */
    private servicesService = inject(ServicesService)

    /**
     * Angular zone to call Router navigation inside the zone.
     * @private
     */
    private zone = inject(NgZone)

    /**
     * PrimeNG message service used to display feedback messages in UI.
     * @private
     */
    private messageService = inject(MessageService)

    /**
     * Angular router service used to navigate when zones table filtering changes.
     * @private
     */
    private router = inject(Router)

    /**
     * Service used for RxJS subject cleanup.
     * @private
     */
    private destroyRef = inject(DestroyRef)

    /**
     * Configures the breadcrumbs for the component.
     * @protected
     */
    protected readonly breadcrumbs: MenuItem[] = [{ label: 'DNS' }, { label: 'Zone Transfers' }]

    /**
     * Collection of zone transfers fetched from backend.
     */
    zoneTransfers: ZoneTransferState[] = []

    /**
     * Total count of zone transfers fetched from backend.
     */
    zoneTransfersTotal: number = 0

    /**
     * Flag stating whether zone transfers table data is loading or not.
     */
    zoneTransfersLoading: boolean = false

    /**
     * Keeps number of zone transfers per page in the zone transfers table.
     */
    zoneTransfersRows: number = 10

    /**
     * Key to be used in browser storage for keeping zones table state.
     * @private
     */
    private readonly _zoneTransfersTableStateStorageKey = 'zone-transfers-table-state'

    /**
     * PrimeNG table component containing list of all zones.
     */
    @ViewChild('zoneTransfersTable') zoneTransfersTable!: Table

    /**
     * Reference to an enum so it could be used in the HTML template.
     * @protected
     */
    protected readonly ZoneTransferSortField = ZoneTransferSortField

    /**
     * RxJS Subject used for filtering zone transfers based on UI filtering form inputs.
     * @private
     */
    private _zoneTransfersTableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * Zone types values used for the UI filter dropdown options.
     * @protected
     */
    protected readonly zoneTransferStatuses: string[] = ['started', 'connected', 'completed', 'message']

    /**
     * Collection of machines fetched from backend.
     *
     * They are listed in the UI filter dropdowns for filtering by primary and secondary.
     */
    machines: MachinesDirectoryEntry[] = []

    /**
     * Flag indicating that the machines are loading.
     *
     * It is used in the dropdowns for filtering by primary and secondary to
     * indicate that the list of machines is being loaded.
     */
    machinesLoading: boolean = false

    /**
     * Reference to the function so it can be used in html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Object containing supported zone transfers filters which values are provided via URL deep-link.
     * The properties of this object correspond to queryParam keys.
     * Values of this object describe:
     * - filter type (numeric, enum, string or boolean)
     * - filter matchMode (contains, equals) which corresponds to PrimeNG table filter metadata
     * - accepted enum values for enum type of filters
     * - array type; when set to true it means that the filter may use more than one value.
     */
    protected readonly supportedQueryParamFilters: {
        [k: string]: {
            type: 'numeric' | 'enum' | 'string' | 'boolean'
            matchMode: 'contains' | 'equals'
            enumValues?: string[]
            arrayType?: boolean
        }
    } = {
        zoneTransferStatus: {
            type: 'enum',
            matchMode: 'equals',
            enumValues: this.zoneTransferStatuses,
            arrayType: true,
        },
        text: { type: 'string', matchMode: 'contains' },
        includeLocal: { type: 'boolean', matchMode: 'equals' },
        serverMachineId: { type: 'numeric', matchMode: 'equals' },
        clientMachineId: { type: 'numeric', matchMode: 'equals' },
        zoneSerial: { type: 'string', matchMode: 'contains' },
    }

    /**
     * Component lifecycle hook which inits the component.
     *
     * It initializes the supported query param filters for the zone transfers table.
     * It restores the rows per page count for the zone transfers table from the state stored
     * in the browser storage. Finally, it uses the REST API to fetch the list of machines
     * for filtering by primary and/or secondary and the list of the zone transfers.
     */
    ngOnInit(): void {
        this._restoreZoneTransfersTableRowsPerPage()

        // Fetch machines for filtering by XFR primary and secondary.
        this.machinesLoading = true
        lastValueFrom(this.servicesService.getMachinesDirectory([Daemon.NameEnum.Named]))
            .then((machines) => {
                this.machines = machines.items ?? []
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error retrieving machines',
                    detail: 'Failed to retrieve machines information for filtering by primary and secondary: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.machinesLoading = false
            })

        // Subscribe to the filter input changes and refresh the table according
        // to the filter values.
        this._zoneTransfersTableFilter$
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
                this.zone.run(() =>
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.zoneTransfersTable) })
                )
            })
    }

    /**
     * Lazily loads paged zone transfers data from backend.
     *
     * @param event PrimeNG TableLazyLoadEvent with metadata about table pagination
     * and filtering.
     */
    onLazyLoadZoneTransfers(event: TableLazyLoadEvent) {
        this.zoneTransfersLoading = true
        this.cd.detectChanges() // in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        lastValueFrom(
            this.dnsService.getZoneTransferStates(
                event?.first ?? 0,
                event?.rows ?? 10,
                (event?.filters?.zoneTransferStatus as FilterMetadata)?.value ?? [],
                (event?.filters?.text as FilterMetadata)?.value ?? undefined,
                (event?.filters?.serverMachineId as FilterMetadata)?.value ?? undefined,
                (event?.filters?.clientMachineId as FilterMetadata)?.value ?? undefined,
                (event?.filters?.zoneSerial as FilterMetadata)?.value ?? undefined,
                (event?.filters?.includeLocal as FilterMetadata)?.value || undefined,
                ...convertSortingFields<ZoneTransferSortField>(event)
            )
        )
            .then((resp) => {
                this.zoneTransfers = resp?.items ?? []
                this.zoneTransfersTotal = resp?.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error retrieving zone transfers',
                    detail: 'Retrieving zone transfers information failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.zoneTransfersLoading = false
            })
    }

    /**
     * Emits next value and filterConstraint for the zones table's filter,
     * which in the end will result in applying the filter on the table's data.
     *
     * @param value value of the filter.
     * @param filterConstraint filter field which will be filtered.
     */
    filterZoneTransfersTable(value: any, filterConstraint: FilterMetadata): void {
        this._zoneTransfersTableFilter$.next({ value, filterConstraint })
    }

    /**
     * Clears a value for given zone transfer table filter constraint and reloads the
     * table with no filtering.
     *
     * @param filterConstraint filter constraint to be cleared.
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.zone.run(() =>
            this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.zoneTransfersTable) })
        )
    }

    /**
     * Stores rows per page count for the zone transfers table in user browser storage.
     */
    storeZoneTransfersTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.zoneTransfersTable?.getStorage()
        storage?.setItem(this._zoneTransfersTableStateStorageKey, JSON.stringify(state))
    }

    /**
     * Restores only rows per page count for the zones table from the state stored in user browser storage.
     * @private
     */
    private _restoreZoneTransfersTableRowsPerPage() {
        const stateString = localStorage.getItem(this._zoneTransfersTableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.zoneTransfersRows = state.rows ?? 10
        }
    }
}
