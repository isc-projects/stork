import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable } from 'rxjs'

import { MenuItem } from 'primeng/api'

import { GeneralService } from './backend/api/api'
import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'
import { SettingService } from './setting.service'
import { ServerDataService } from './server-data.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit {
    storkVersion = 'unknown'
    storkBuildDate = 'unknown'
    currentUser = null
    loadingInProgress = new Observable()
    userMenuItems: MenuItem[]

    menuItems: MenuItem[]

    breadcrumbItems: MenuItem[]

    constructor(
        private router: Router,
        private serverData: ServerDataService,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService,
        private settingSvc: SettingService
    ) {
        this.initMenus()

        this.breadcrumbItems = [{ label: 'Categories' }]

        this.loadingInProgress = this.loadingService.getState()
    }

    initMenus() {
        this.userMenuItems = [
            {
                label: 'Profile',
                icon: 'fa fa-cog',
                routerLink: '/profile',
            },
        ]

        this.menuItems = [
            {
                label: 'DHCP',
                visible: false,
                items: [
                    {
                        label: 'Dashboard',
                        icon: 'fa fa-tachometer-alt',
                        routerLink: '/dashboard',
                    },
                    {
                        label: 'Host Reservations',
                        icon: 'fa fa-laptop',
                        routerLink: '/dhcp/hosts',
                    },
                    {
                        label: 'Subnets',
                        icon: 'fa fa-project-diagram',
                        routerLink: '/dhcp/subnets',
                    },
                    {
                        label: 'Shared Networks',
                        icon: 'fa fa-network-wired',
                        routerLink: '/dhcp/shared-networks',
                    },
                ],
            },
            {
                label: 'Services',
                items: [
                    {
                        label: 'Kea Apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/kea/all',
                    },
                    {
                        label: 'BIND 9 Apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/bind9/all',
                    },
                    {
                        label: 'Machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/all',
                    },
                    {
                        label: 'Grafana',
                        icon: 'pi pi-chart-line',
                        url: '',
                        visible: false,
                    },
                ],
            },
            {
                label: 'Monitoring',
                items: [
                    {
                        label: 'Events',
                        icon: 'fa fa-calendar-times',
                        routerLink: '/events',
                    },
                ],
            },
            {
                label: 'Configuration',
                items: [
                    {
                        label: 'Users',
                        visible: false,
                        icon: 'fa fa-user',
                        routerLink: '/users',
                    },
                    {
                        label: 'Settings',
                        icon: 'fa fa-cog',
                        routerLink: '/settings',
                    },
                ],
            },
            {
                label: 'Help',
                items: [
                    {
                        label: 'Stork Manual',
                        icon: 'fa fa-book',
                        url: '/assets/arm/index.html',
                        target: 'blank',
                    },
                    {
                        label: 'Stork API Docs (SwaggerUI)',
                        icon: 'fa fa-code',
                        routerLink: '/swagger-ui',
                    },
                    {
                        label: 'Stork API Docs (Redoc)',
                        icon: 'fa fa-code',
                        url: '/api/docs',
                        target: 'blank',
                    },
                    {
                        label: 'BIND 9 Manual',
                        icon: 'fa fa-book',
                        url: 'https://downloads.isc.org/isc/bind9/cur/9.16/doc/arm/Bv9ARM.html',
                        target: 'blank',
                    },
                    {
                        label: 'Kea Manual',
                        icon: 'fa fa-book',
                        url: 'https://kea.readthedocs.io/',
                        target: 'blank',
                    },
                ],
            },
        ]
    }

    getMenuItem(name) {
        for (const menuItem of this.menuItems) {
            if (menuItem['label'] === name) {
                return menuItem
            } else {
                for (const subMenu of menuItem.items) {
                    if (subMenu['label'] === name) {
                        return subMenu
                    }
                }
            }
        }
    }

    ngOnInit() {
        this.generalApi.getVersion().subscribe((data) => {
            this.storkVersion = data.version
            this.storkBuildDate = data.date
        })

        this.auth.currentUser.subscribe((x) => {
            this.currentUser = x
            const menuItem = this.getMenuItem('Users')
            if (this.auth.superAdmin()) {
                // super admin can see Configuration/Users menu
                menuItem['visible'] = true
            } else {
                menuItem['visible'] = false
            }

            // Only get the stats and settings when the user is logged in.
            if (this.auth.currentUserValue) {
                this.serverData.getAppsStats().subscribe((data) => {
                    // if there are Kea apps then show Kea related menu items
                    // otherwise hide them
                    const dhcpMenuItem = this.getMenuItem('DHCP')
                    const keaAppsMenuItem = this.getMenuItem('Kea Apps')
                    if (data.keaAppsTotal && data.keaAppsTotal > 0) {
                        dhcpMenuItem.visible = true
                        keaAppsMenuItem['visible'] = true
                    } else {
                        dhcpMenuItem.visible = false
                        keaAppsMenuItem['visible'] = false
                    }
                    // if there are BIND 9 apps then show BIND 9 related menu items
                    // otherwise hide them
                    const bind9AppsMenuItem = this.getMenuItem('BIND 9 Apps')
                    if (data.bind9AppsTotal && data.bind9AppsTotal > 0) {
                        bind9AppsMenuItem['visible'] = true
                    } else {
                        bind9AppsMenuItem['visible'] = false
                    }
                })

                // If Grafana url is not empty, we need to make
                // Services.Grafana menu choice visible and set it's url.
                // Otherwise we need to make sure it's not visible.
                this.settingSvc.getSettings().subscribe((data) => {
                    const grafanaUrl = data['grafana_url']

                    const grafanaMenuItem = this.getMenuItem('Grafana')

                    if (grafanaUrl && grafanaUrl !== '') {
                        grafanaMenuItem['visible'] = true
                        grafanaMenuItem['url'] = grafanaUrl
                    } else {
                        grafanaMenuItem['visible'] = false
                    }
                })
            }
        })
    }

    signOut() {
        this.router.navigate(['/logout'])
    }
}
