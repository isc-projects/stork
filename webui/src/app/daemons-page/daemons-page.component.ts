import { Component, OnDestroy, OnInit, viewChild, ViewChild } from '@angular/core'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'

import { MessageService, MenuItem, ConfirmationService, TableState, PrimeTemplate } from 'primeng/api'

import {
    daemonNameToFriendlyName,
    daemonStatusErred,
    daemonStatusIconClass as daemonStatusIconClassFn,
    daemonStatusIconTooltip as daemonStatusIconTooltipFn,
} from '../utils'
import { AnyDaemon, ServicesService } from '../backend'
import { Table, TableLazyLoadEvent, TableModule } from 'primeng/table'
import { Menu } from 'primeng/menu'
import { distinctUntilChanged, finalize, map } from 'rxjs/operators'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { Router, RouterLink } from '@angular/router'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { Panel } from 'primeng/panel'
import { NgIf } from '@angular/common'
import { Tag } from 'primeng/tag'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { FloatLabel } from 'primeng/floatlabel'
import { MultiSelect } from 'primeng/multiselect'
import { FormsModule } from '@angular/forms'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputText } from 'primeng/inputtext'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { DaemonTabComponent } from '../daemon-tab/daemon-tab.component'
import { Tooltip } from 'primeng/tooltip'
import { EntityLinkComponent } from '../entity-link/entity-link.component'

/**
 * Sets boolean flag indicating if there are communication errors with
 * the daemon.
 */
function setDaemonStatusErred(daemon: AnyDaemon & { statusErred?: boolean }) {
    if (daemon) {
        daemon.statusErred = daemon.active && daemonStatusErred(daemon as any)
    }
}

@Component({
    selector: 'app-daemons-page',
    templateUrl: './daemons-page.component.html',
    styleUrls: ['./daemons-page.component.sass'],
    imports: [
        ConfirmDialog,
        BreadcrumbsComponent,
        TabViewComponent,
        Button,
        ManagedAccessDirective,
        Menu,
        TableModule,
        Panel,
        NgIf,
        Tag,
        HelpTipComponent,
        PrimeTemplate,
        FloatLabel,
        MultiSelect,
        FormsModule,
        IconField,
        InputIcon,
        InputText,
        RouterLink,
        VersionStatusComponent,
        DaemonTabComponent,
        Tooltip,
        EntityLinkComponent,
    ],
})
export class DaemonsPageComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG Table with daemons list.
     */
    @ViewChild('table') daemonsTable: Table

    /**
     * Daemon menu component.
     */
    @ViewChild('daemonMenu') daemonMenu: Menu

    breadcrumbs: MenuItem[] = []

    // daemons table
    daemons: (AnyDaemon & { statusErred?: boolean })[] = []
    totalDaemons: number
    daemonMenuItems: MenuItem[]
    dataLoading: boolean

    /**
     * Asynchronously provides a daemon entity based on given daemon ID.
     * @param daemonId daemon ID
     */
    daemonProvider: (id: number) => Promise<AnyDaemon> = (daemonId: number) => {
        this.dataLoading = true
        return lastValueFrom(
            this.servicesApi.getDaemon(daemonId).pipe(
                map((data) => {
                    setDaemonStatusErred(data)
                    return data
                }),
                finalize(() => (this.dataLoading = false))
            )
        )
    }

    constructor(
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private confirmService: ConfirmationService,
        private router: Router
    ) {}

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions: Subscription

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     * Emits next value and filterConstraint for the table's filter,
     * which in the end will result in applying the filter on the table's data.
     * @param value value of the filter
     * @param filterConstraint filter field which will be filtered
     */
    filterTable(value: any, filterConstraint: FilterMetadata): void {
        this._tableFilter$.next({ value, filterConstraint })
    }

    /** Returns a tab title - formatted daemon name. */
    tabTitleProvider(daemon: AnyDaemon): string {
        return daemonNameToFriendlyName(daemon.name)
    }

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.daemonsTable?.clearFilterValues()
        this.router.navigate([])
    }

    ngOnInit() {
        this.breadcrumbs = [{ label: 'Services' }, { label: 'Daemons' }]

        this.daemons = []
        this.daemonMenuItems = [
            {
                label: 'Refresh',
                id: 'refresh-single-daemon',
                icon: 'pi pi-refresh',
            },
        ]

        this._restoreTableRowsPerPage()

        this._subscriptions = this._tableFilter$
            .pipe(
                map((f) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                debounceTime(300),
                distinctUntilChanged()
            )
            .subscribe((f) => {
                // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                // so it's value must be set according to UI columnFilter value.
                f.filterConstraint.value = f.value
                this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.daemonsTable) })
            })
    }

    ngOnDestroy() {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Function called by the table data loader. Accepts the pagination event.
     */
    loadDaemons(event: TableLazyLoadEvent) {
        this.dataLoading = true

        // ToDo: Uncaught promise
        // If any HTTP exception will be thrown then the promise
        // fails, but a user doesn't get any message, popup, log.
        lastValueFrom(
            this.servicesApi.getDaemons(
                event.first,
                event.rows,
                (event.filters['text'] as FilterMetadata)?.value || null,
                (event.filters['daemons'] as FilterMetadata)?.value ?? null
            )
        )
            .then((data) => {
                this.daemons = data.items ?? []
                this.totalDaemons = data.total ?? 0
                this.daemons.forEach((daemon) => setDaemonStatusErred(daemon))
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    /**
     * TabView component which is a view child.
     */
    tabView = viewChild(TabViewComponent)

    /**
     * Callback called on click on the daemon menu button.
     *
     * @param event click event
     * @param daemonId daemon identifier
     */
    showDaemonMenu(event: Event, daemonId: number) {
        // connect method to refresh machine state
        this.daemonMenuItems[0].command = () => {
            this.tabView()?.onUpdateTabEntity(daemonId)
        }

        this.daemonMenu.toggle(event)
    }

    /** Callback called on click the refresh daemon list button. */
    refreshDaemonsList() {
        this.loadDaemons(this.daemonsTable?.createLazyLoadMetadata())
    }

    /**
     * Sends a request to the server to re-synchronize Kea configs.
     *
     * Clearing the config hashes causes the server to fetch and update
     * Kea configurations in the Stork database.
     */
    onSyncKeaConfigs(): void {
        this.confirmService.confirm({
            message:
                'This operation instructs the server to fetch the configurations from all Kea servers' +
                ' and update them in the Stork database. Use it if you suspect that the configuration' +
                ' information differs between Kea and Stork. This operation should be harmless and typically' +
                ' causes only some additional overhead to populate the fetched data. Populating the data can' +
                ' take some time, depending on the puller-interval settings and the availability of the Kea servers.',
            header: 'Resynchronize Kea Configs',
            icon: 'pi pi-exclamation-triangle',
            acceptLabel: 'Continue',
            rejectLabel: 'Cancel',
            rejectButtonProps: {
                text: true,
                icon: 'pi pi-times',
            },
            acceptButtonProps: {
                icon: 'pi pi-check',
            },
            accept: () => {
                // User confirmed. Clear the hashes in the server.
                this.servicesApi
                    .deleteKeaDaemonConfigHashes()
                    .toPromise()
                    .then(() => {
                        this.msgSrv.add({
                            severity: 'success',
                            summary: 'Request to resynchronize sent',
                            detail:
                                'Successfully sent the request to the server to resynchronize' +
                                ' Kea configurations in the Stork server. It may take a while' +
                                ' before it takes effect.',
                            life: 10000,
                        })
                    })
                    .catch(() => {
                        this.msgSrv.add({
                            severity: 'error',
                            summary: 'Request to resynchronize failed',
                            detail:
                                'The request to resynchronize Kea configurations in Stork failed' +
                                ' due to an internal server error. You can try again to see' +
                                ' if the error goes away.',
                            life: 10000,
                        })
                    })
            },
        })
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
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.daemonsTable) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'daemons-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.daemonsTable?.getStorage()
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
     * Returns an CSS icon name to indicate the daemon status.
     */
    daemonStatusIconClass(daemon: AnyDaemon) {
        return daemonStatusIconClassFn(daemon as any)
    }

    /**
     * Returns a tooltip that should be assigned to the status icon.
     */
    daemonStatusIconTooltip(daemon: AnyDaemon) {
        return daemonStatusIconTooltipFn(daemon as any)
    }

    /**
     * Handler called when the refresh button has been clicked.
     */
    onRefreshDaemon(daemonId: number) {
        this.tabView()?.onUpdateTabEntity(daemonId)
    }
}
