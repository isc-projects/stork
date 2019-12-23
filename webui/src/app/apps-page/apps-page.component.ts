import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'

function htmlizeExtVersion(app) {
    if (app.details.extendedVersion) {
        app.details.extendedVersion = app.details.extendedVersion.replace(/\n/g, '<br>')
    }
    for (const d of app.details.daemons) {
        if (d.extendedVersion) {
            d.extendedVersion = d.extendedVersion.replace(/\n/g, '<br>')
        }
    }
}

function capitalize(txt) {
    return txt.charAt(0).toUpperCase() + txt.slice(1)
}

@Component({
    selector: 'app-apps-page',
    templateUrl: './apps-page.component.html',
    styleUrls: ['./apps-page.component.sass'],
})
export class AppsPageComponent implements OnInit {
    appType: string
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

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private loadingService: LoadingService
    ) {}

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
        console.info('addAppTab', app)
        this.openedApps.push({
            app,
            activeDaemonTabIdx: 0,
        })
        const capAppType = capitalize(app.type)
        this.tabs.push({
            label: `${app.id}. ${capAppType}@${app.machine.address}`,
            routerLink: '/apps/' + this.appType + '/' + app.id,
        })
    }

    ngOnInit() {
        this.appType = this.route.snapshot.params.srv
        this.tabs = [{ label: capitalize(this.appType) + ' Apps', routerLink: '/apps/' + this.appType + '/all' }]

        this.apps = []
        this.appMenuItems = [
            {
                label: 'Refresh',
                icon: 'pi pi-refresh',
            },
        ]

        this.openedApps = []

        this.route.paramMap.subscribe((params: ParamMap) => {
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
                        data => {
                            htmlizeExtVersion(data)
                            this.addAppTab(data)
                            this.switchToTab(this.tabs.length - 1)
                        },
                        err => {
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
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        this.servicesApi.getApps(event.first, event.rows, text, this.appType).subscribe(data => {
            this.apps = data.items
            this.totalApps = data.total
            for (const s of this.apps) {
                htmlizeExtVersion(s)
            }
        })
    }

    keyDownFilterText(appsTable, event) {
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
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'App refreshed',
                    detail: 'Refreshing succeeded.',
                })

                htmlizeExtVersion(data)

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
                        break
                    }
                }
            },
            err => {
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

    sortKeaDaemonsByImportance(app) {
        const daemonMap = []
        for (const d of app.details.daemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [
            ['dhcp4', 'DHCPv4'],
            ['dhcp6', 'DHCPv6'],
            ['d2', 'DDNS'],
            ['ca', 'CA'],
            ['netconf', 'NETCONF'],
        ]
        const daemons = []
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemons.push(daemonMap[dm[0]])
            }
        }
        return daemons
    }
}
