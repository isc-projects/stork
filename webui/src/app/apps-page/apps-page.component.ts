import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'
import { BehaviorSubject } from 'rxjs'

import { MessageService, MenuItem } from 'primeng/api'

import { daemonStatusErred } from '../utils'
import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'

function htmlizeExtVersion(app) {
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
export class AppsPageComponent implements OnInit {
    breadcrumbs = [{ label: 'Services' }, { label: 'Apps' }]

    appType = ''
    // apps table
    apps: any[]
    totalApps: number
    appMenuItems: MenuItem[]

    // action panel
    filterText = ''

    // machine tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    openedApps: any
    appTab: any

    refreshedAppTab = new BehaviorSubject(this.appTab)

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private loadingService: LoadingService
    ) {}

    getAppsLabel() {
        if (this.appType === 'bind9') {
            return 'BIND 9 Apps'
        } else {
            return ' Kea Apps'
        }
    }

    switchToTab(index) {
        if (this.activeTabIdx === index) {
            return
        }
        this.activeTabIdx = index
        this.activeItem = this.tabs[index]

        if (index > 0) {
            this.appTab = this.openedApps[index - 1]
        }
    }

    addAppTab(app) {
        this.openedApps.push({
            app,
        })
        this.tabs.push({
            label: `[${app.id}]@${app.machine.address}`,
            routerLink: '/apps/' + this.appType + '/' + app.id,
        })
    }

    ngOnInit() {
        this.route.paramMap.subscribe((params) => {
            const newAppType = params.get('appType')

            if (newAppType !== this.appType) {
                this.appType = newAppType
                this.breadcrumbs[1]['label'] = this.getAppsLabel()

                this.tabs = [{ label: 'All', routerLink: '/apps/' + this.appType + '/all' }]

                this.apps = []
                this.appMenuItems = [
                    {
                        label: 'Refresh',
                        id: 'refresh-single-app',
                        icon: 'pi pi-refresh',
                    },
                ]

                this.openedApps = []

                this.loadApps({ first: 0, rows: 10 })
            }

            const appIdStr = params.get('id')
            if (appIdStr === 'all') {
                this.switchToTab(0)
            } else {
                const appId = parseInt(appIdStr, 10)

                let found = false
                // if tab for this app is already opened then switch to it
                for (let idx = 0; idx < this.openedApps.length; idx++) {
                    const s = this.openedApps[idx].app
                    if (s.id === appId) {
                        console.info('found opened app', idx)
                        this.switchToTab(idx + 1)
                        found = true
                    }
                }

                // if tab is not opened then search for list of apps if the one is present there,
                // if so then open it in new tab and switch to it
                if (!found) {
                    for (const s of this.apps) {
                        if (s.id === appId) {
                            console.info('found app in the list, opening it')
                            this.addAppTab(s)
                            this.switchToTab(this.tabs.length - 1)
                            found = true
                            break
                        }
                    }
                }

                // if app is not loaded in list fetch it individually
                if (!found) {
                    console.info('fetching app')
                    this.servicesApi.getApp(appId).subscribe(
                        (data) => {
                            if (data.type !== this.appType) {
                                this.msgSrv.add({
                                    severity: 'error',
                                    summary: 'Cannot find app',
                                    detail: 'Cannot find app with ID ' + appId,
                                    life: 10000,
                                })
                                this.router.navigate(['/apps/' + this.appType + '/all'])
                                return
                            }

                            htmlizeExtVersion(data)
                            setDaemonStatusErred(data)
                            this.addAppTab(data)
                            this.switchToTab(this.tabs.length - 1)
                        },
                        (err) => {
                            let msg = err.statusText
                            if (err.error && err.error.message) {
                                msg = err.error.message
                            }
                            this.msgSrv.add({
                                severity: 'error',
                                summary: 'Cannot get app',
                                detail: 'Getting app with ID ' + appId + ' erred: ' + msg,
                                life: 10000,
                            })
                            this.router.navigate(['/apps/' + this.appType + '/all'])
                        }
                    )
                }
            }
        })
    }

    loadApps(event) {
        if (this.appType === '') {
            // appType has not been set yet so do not load anything
            return
        }
        let text
        if (event.filters && event.filters.text) {
            text = event.filters.text.value
        }

        this.servicesApi.getApps(event.first, event.rows, text, this.appType).subscribe((data) => {
            this.apps = data.items
            this.totalApps = data.total
            for (const s of this.apps) {
                htmlizeExtVersion(s)
                setDaemonStatusErred(s)
            }
        })
    }

    keyUpFilterText(appsTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            appsTable.filter(this.filterText, 'text', 'equals')
        }
    }

    closeTab(event, idx) {
        this.openedApps.splice(idx - 1, 1)
        this.tabs.splice(idx, 1)
        if (this.activeTabIdx === idx) {
            this.switchToTab(idx - 1)
            if (idx - 1 > 0) {
                this.router.navigate(['/apps/' + this.appType + '/' + this.appTab.app.id])
            } else {
                this.router.navigate(['/apps/' + this.appType + '/all'])
            }
        } else if (this.activeTabIdx > idx) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    _refreshAppState(app) {
        this.servicesApi.getApp(app.id).subscribe(
            (data) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'App refreshed',
                    detail: 'Refreshing succeeded.',
                })

                htmlizeExtVersion(data)
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
                        this.refreshedAppTab.next(this.appTab)
                        break
                    }
                }
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Getting app state erred',
                    detail: 'Getting state of app erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    showAppMenu(event, appMenu, app) {
        appMenu.toggle(event)

        // connect method to refresh machine state
        this.appMenuItems[0].command = () => {
            this._refreshAppState(app)
        }
    }

    onRefreshApp(event) {
        this._refreshAppState(this.appTab.app)
    }

    refreshAppsList(appsTable) {
        appsTable.onLazyLoad.emit(appsTable.createLazyLoadMetadata())
    }
}
