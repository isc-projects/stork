import { Component, OnDestroy, OnInit, viewChild, ViewChild } from '@angular/core'
import { debounceTime, lastValueFrom, Subject, Subscription } from 'rxjs'

import { MessageService, MenuItem, ConfirmationService, TableState } from 'primeng/api'

import { daemonStatusErred } from '../utils'
import { ServicesService } from '../backend'
import { App } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { Menu } from 'primeng/menu'
import { distinctUntilChanged, finalize, map } from 'rxjs/operators'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { Router } from '@angular/router'
import { TabViewComponent } from '../tab-view/tab-view.component'

/**
 * Replaces the newlines in the versions with the HTML-compatible line breaks.
 * @param app Application
 */
function htmlizeExtVersion(app: App) {
    if (app.details.extendedVersion) {
        app.details.extendedVersion = app.details.extendedVersion.replace(/\n/g, '<br>')
    }
    if (app.details.daemons) {
        for (const d of app.details.daemons) {
            if (d.extendedVersion) {
                d.extendedVersion = d.extendedVersion.replace(/\n/g, '<br>')
            }
        }
    }
}

/**
 * Sets boolean flag indicating if there are communication errors with
 * daemons belonging to the app.
 *
 * @param app app for which the communication status with the daemons
 *            should be updated.
 */
function setDaemonStatusErred(app) {
    if (app.details.daemons) {
        for (const d of app.details.daemons) {
            d.statusErred = d.active && daemonStatusErred(d)
        }
    }
}

@Component({
    selector: 'app-apps-page',
    templateUrl: './apps-page.component.html',
    styleUrls: ['./apps-page.component.sass'],
})
export class AppsPageComponent implements OnInit, OnDestroy {
    /**
     * PrimeNG Table with apps list.
     */
    @ViewChild('table') appsTable: Table

    /**
     * Application menu component.
     */
    @ViewChild('appMenu') appMenu: Menu

    breadcrumbs: MenuItem[] = []

    // apps table
    apps: App[] = []
    totalApps: number
    appMenuItems: MenuItem[]
    dataLoading: boolean

    /**
     * Asynchronously provides an App entity based on given App ID.
     * @param appID application ID
     */
    appProvider: (id: number) => Promise<App> = (appID: number) => {
        this.dataLoading = true
        return lastValueFrom(
            this.servicesApi.getApp(appID).pipe(
                map((data) => {
                    htmlizeExtVersion(data)
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

    /**
     * Clears the PrimeNG table state (filtering, pagination are reset).
     */
    clearTableState() {
        this.appsTable?.clear()
        this.router.navigate([])
    }

    ngOnInit() {
        this.breadcrumbs = [{ label: 'Services' }, { label: 'Apps' }]

        this.apps = []
        this.appMenuItems = [
            {
                label: 'Refresh',
                id: 'refresh-single-app',
                icon: 'pi pi-refresh',
            },
        ]

        this._restoreTableRowsPerPage()

        this._subscriptions = this._tableFilter$
            .pipe(
                map((f) => {
                    return { ...f, value: f.value ?? null }
                }),
                debounceTime(300),
                distinctUntilChanged(),
                map((f) => {
                    f.filterConstraint.value = f.value
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.appsTable) })
                })
            )
            .subscribe()
    }

    ngOnDestroy() {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Function called by the table data loader. Accepts the pagination event.
     */
    loadApps(event: TableLazyLoadEvent) {
        console.log('loadApps', event, Date.now())
        this.dataLoading = true

        // ToDo: Uncaught promise
        // If any HTTP exception will be thrown then the promise
        // fails, but a user doesn't get any message, popup, log.
        lastValueFrom(
            this.servicesApi.getApps(
                event.first,
                event.rows,
                (event.filters['text'] as FilterMetadata)?.value || null,
                (event.filters['apps'] as FilterMetadata)?.value ?? null
            )
        )
            .then((data) => {
                this.apps = data.items ?? []
                this.totalApps = data.total ?? 0
                for (const s of this.apps) {
                    htmlizeExtVersion(s)
                    setDaemonStatusErred(s)
                }
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
     * Callback called on click on the application menu button.
     *
     * @param event
     * @param appID
     */
    showAppMenu(event: Event, appID: number) {
        // connect method to refresh machine state
        this.appMenuItems[0].command = () => {
            this.tabView()?.onUpdateTabEntity(appID)
        }

        this.appMenu.toggle(event)
    }

    /** Callback called on click the refresh application list button. */
    refreshAppsList() {
        this.loadApps(this.appsTable?.createLazyLoadMetadata())
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

    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.appsTable) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'apps-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.appsTable?.getStorage()
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
}
