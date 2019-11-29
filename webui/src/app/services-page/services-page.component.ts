import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'

function htmlizeExtVersion(service) {
    if (service.details.extendedVersion) {
        service.details.extendedVersion = service.details.extendedVersion.replace(/\n/g, '<br>')
    }
    for (const d of service.details.daemons) {
        if (d.extendedVersion) {
            d.extendedVersion = d.extendedVersion.replace(/\n/g, '<br>')
        }
    }
}

@Component({
    selector: 'app-services-page',
    templateUrl: './services-page.component.html',
    styleUrls: ['./services-page.component.sass'],
})
export class ServicesPageComponent implements OnInit {
    serviceType: string
    // services table
    services: any[]
    totalServices: number
    serviceMenuItems: MenuItem[]

    // action panel
    filterText = ''

    // machine tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    openedServices: any
    serviceTab: any

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
            this.serviceTab = this.openedServices[index - 1]
        }
    }

    addServiceTab(service) {
        console.info('addServiceTab', service)
        this.openedServices.push({
            service,
            aaa: service.id,
        })
        this.tabs.push({
            label: service.id,
            routerLink: '/services/' + this.serviceType + '/' + service.id,
        })
    }

    ngOnInit() {
        this.serviceType = this.route.snapshot.params.srv
        this.tabs = [{ label: 'Services', routerLink: '/services/' + this.serviceType + '/all' }]

        this.services = []
        this.serviceMenuItems = [
            {
                label: 'Refresh',
                icon: 'pi pi-refresh',
            },
        ]

        this.openedServices = []

        this.route.paramMap.subscribe((params: ParamMap) => {
            const serviceIdStr = params.get('id')
            if (serviceIdStr === 'all') {
                this.switchToTab(0)
            } else {
                const serviceId = parseInt(serviceIdStr, 10)

                let found = false
                // if tab for this service is already opened then switch to it
                for (let idx = 0; idx < this.openedServices.length; idx++) {
                    const s = this.openedServices[idx].service
                    if (s.id === serviceId) {
                        console.info('found opened service', idx)
                        this.switchToTab(idx + 1)
                        found = true
                    }
                }

                // if tab is not opened then search for list of services if the one is present there,
                // if so then open it in new tab and switch to it
                if (!found) {
                    for (const s of this.services) {
                        if (s.id === serviceId) {
                            console.info('found service in the list, opening it')
                            this.addServiceTab(s)
                            this.switchToTab(this.tabs.length - 1)
                            found = true
                            break
                        }
                    }
                }

                // if service is not loaded in list fetch it individually
                if (!found) {
                    console.info('fetching service')
                    this.servicesApi.getService(serviceId).subscribe(
                        data => {
                            htmlizeExtVersion(data)
                            this.addServiceTab(data)
                            this.switchToTab(this.tabs.length - 1)
                        },
                        err => {
                            let msg = err.statusText
                            if (err.error && err.error.message) {
                                msg = err.error.message
                            }
                            this.msgSrv.add({
                                severity: 'error',
                                summary: 'Cannot get service',
                                detail: 'Getting service with ID ' + serviceId + ' erred: ' + msg,
                                life: 10000,
                            })
                            this.router.navigate(['/services/' + this.serviceType + '/all'])
                        }
                    )
                }
            }
        })
    }

    loadServices(event) {
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        this.servicesApi.getServices(event.first, event.rows, text, this.serviceType).subscribe(data => {
            this.services = data.items
            this.totalServices = data.total
            for (const s of this.services) {
                htmlizeExtVersion(s)
            }
        })
    }

    keyDownFilterText(servicesTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            servicesTable.filter(this.filterText, 'text', 'equals')
        }
    }

    closeTab(event, idx) {
        this.openedServices.splice(idx - 1, 1)
        this.tabs.splice(idx, 1)
        if (this.activeTabIdx === idx) {
            this.switchToTab(idx - 1)
            if (idx - 1 > 0) {
                this.router.navigate(['/services/' + this.serviceType + '/' + this.serviceTab.service.id])
            } else {
                this.router.navigate(['/services/' + this.serviceType + '/all'])
            }
        } else if (this.activeTabIdx > idx) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    _refreshServiceState(service) {
        this.servicesApi.getService(service.id).subscribe(
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Service refreshed',
                    detail: 'Refreshing succeeded.',
                })

                htmlizeExtVersion(data)

                // refresh service in service list
                for (const s of this.services) {
                    if (s.id === data.id) {
                        Object.assign(s, data)
                        break
                    }
                }
                // refresh machine in opened tab if present
                for (const s of this.openedServices) {
                    if (s.service.id === data.id) {
                        Object.assign(s.service, data)
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
                    summary: 'Getting service state erred',
                    detail: 'Getting state of service erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    showServiceMenu(event, serviceMenu, service) {
        serviceMenu.toggle(event)

        // connect method to refresh machine state
        this.serviceMenuItems[0].command = () => {
            this._refreshServiceState(service)
        }
    }

    refreshServiceState(machinesTab) {
        this._refreshServiceState(machinesTab.machine)
    }

    refreshServicesList(servicesTable) {
        servicesTable.onLazyLoad.emit(servicesTable.createLazyLoadMetadata())
    }
}
