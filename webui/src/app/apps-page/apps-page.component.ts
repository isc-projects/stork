import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ActivatedRoute, Router } from '@angular/router'
import { lastValueFrom, Subject, Subscription, tap, debounceTime, distinctUntilChanged } from 'rxjs'

import { MessageService, MenuItem, ConfirmationService } from 'primeng/api'

import { daemonStatusErred, getErrorMessage } from '../utils'
import { ServicesService } from '../backend/api/api'
import { App } from '../backend'
import { Table } from 'primeng/table'
import { Menu } from 'primeng/menu'
import { AppTab } from '../apps'
import { finalize } from 'rxjs/operators'

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
            if (d.active && daemonStatusErred(d)) {
                d.statusErred = true
            } else {
                d.statusErred = false
            }
        }
    }
}

@Component({
    selector: 'app-apps-page',
    templateUrl: './apps-page.component.html',
    styleUrls: ['./apps-page.component.sass'],
})
export class AppsPageComponent implements OnInit, OnDestroy {
    @ViewChild('appsTable') appsTable: Table

    private subscriptions = new Subscription()
    breadcrumbs: MenuItem[] = []

    // apps table
    apps: App[] = []
    totalApps: number
    appMenuItems: MenuItem[]
    dataLoading: boolean

    // app tabs
    openedApps: AppTab[] = []
    appTab: AppTab = null

    refreshedAppTab = new Subject<AppTab>()
    appProvider: (id: number) => Promise<App> = (appId: number) => {
        this.dataLoading = true
        return lastValueFrom(
            this.servicesApi.getApp(appId).pipe(
                tap((data) => {
                    htmlizeExtVersion(data)
                    setDaemonStatusErred(data)
                }),
                finalize(() => (this.dataLoading = false))
            )
        )
    }

    /**
     * Event pipeline to react to changes in the App Types filter dropdown.
     */
    selectedAppTypes$: Subject<string[]> = new Subject()
    /**
     * Storage for the currently selected App Types in the filter dropdown.
     */
    selectedAppTypes: string[] = []

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private confirmService: ConfirmationService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
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

        this.openedApps = []
        // Reload 450ms after the user stops changing filters.  Value chosen by
        // messing around with it until it felt good to me.
        this.subscriptions.add(
            this.selectedAppTypes$.pipe(debounceTime(450), distinctUntilChanged()).subscribe((_) => {
                // Ignore the value *in* the event, just use the existence of the
                // event to cause a refresh.  This avoids rewriting large portions
                // of this component just before the PrimeNG 17 -> 19 upgrade.
                this.refreshAppsList(this.appsTable)
            })
        )

        if (this.appsTable) {
            this.refreshAppsList(this.appsTable)
        }
    }

    /**
     * Function called by the MultiSelect for choosing which app types to show,
     * in order to emit an event which reloads the table.
     */
    updateAppTypesFilter(change_event: string[]) {
        this.selectedAppTypes$.next(change_event)
    }

    /**
     * Function called by the table data loader. Accepts the pagination event.
     */
    loadApps(event) {
        this.dataLoading = true
        let text
        if (event.filters && event.filters.hasOwnProperty('text')) {
            text = event.filters.text.value
        }

        // ToDo: Uncaught promise
        // If any HTTP exception will be thrown then the promise
        // fails, but a user doesn't get any message, popup, log.
        lastValueFrom(this.servicesApi.getApps(event.first, event.rows, text, this.selectedAppTypes))
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
     * Callback called on input event emitted by the filter input box.
     *
     * @param table table on which the filtering will apply
     * @param filterText text value of the filter input
     * @param force force filtering for shorter lookup keywords
     */
    inputFilterText(table: Table, filterText: string, force: boolean = false) {
        if (filterText.length >= 3 || (force && filterText != '')) {
            table.filter(filterText, 'text', 'contains')
        } else if (filterText.length == 0) {
            this.clearFilters(table)
        }
    }

    /** Fetches an application state from the API. */
    _refreshAppState(app: App) {
        this.servicesApi.getApp(app.id).subscribe(
            (data) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'App refreshed',
                    detail: 'Refreshing succeeded.',
                })

                htmlizeExtVersion(app)
                setDaemonStatusErred(data)

                // refresh app in app list
                for (const s of this.apps) {
                    if (s.id === data.id) {
                        Object.assign(s, data)
                        break
                    }
                }
                // refresh machine in opened tab if present
                for (const s of this.openedApps) {
                    if (s.app.id === data.id) {
                        Object.assign(s.app, data)
                        // Notify the child component about the update.
                        this.appTab = { ...s }
                        this.refreshedAppTab.next(this.appTab)
                        break
                    }
                }
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Error getting app state',
                    detail: 'Error getting state of app: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /** Callback called on click on the application menu button. */
    showAppMenu(event: PointerEvent, appMenu: Menu, app: App) {
        appMenu.toggle(event)

        // connect method to refresh machine state
        this.appMenuItems[0].command = () => {
            this._refreshAppState(app)
        }
    }

    /** Callback called on click the refresh button. */
    onRefreshApp() {
        this._refreshAppState(this.appTab.app)
    }

    /** Callback called on click the refresh application list button. */
    refreshAppsList(appsTable) {
        appsTable.onLazyLoad.emit(appsTable.createLazyLoadMetadata())
    }

    /**
     * Modifies an active tab's label after renaming an app.
     *
     * This function is invoked when an app is renamed in a child
     * component, i.e. kea-app-tab or bind9-app-tab. As a result,
     * the label of the currently selected tab is changed to the
     * new app name.
     *
     * @param event holds new app name.
     */
    onRenameApp(event) {
        // TODO: update the impl
        // if (this.activeTabIdx > 0) {
        //     this.tabs[this.activeTabIdx].label = event
        // }
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
     * Clears filtering on given table.
     *
     * @param table table where filtering is to be cleared
     */
    clearFilters(table: Table) {
        table.filter(null, 'text', 'contains')
        this.selectedAppTypes = []
        this.refreshAppsList(this.appsTable)
    }
}
