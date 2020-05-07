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
    title = 'Stork'
    storkVersion = 'unknown'
    storkBuildDate = 'unknown'
    currentUser = null
    loadingInProgress = new Observable()
    userMenuItems: MenuItem[]

    menuItems: MenuItem[]

    constructor(
        private router: Router,
        private serverData: ServerDataService,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService,
        private settingSvc: SettingService
    ) {
        this.initMenus()

        this.auth.currentUser.subscribe((x) => {
            this.currentUser = x
            if (this.auth.superAdmin()) {
                // super admin can see Configuration/Users menu
                this.menuItems[2].items[0]['visible'] = true
            } else {
                this.menuItems[2].items[0]['visible'] = false
            }
        })

        this.serverData.getAppsStats().subscribe((data) => {
            // if there are Kea apps then show Kea related menu items
            // otherwise hide them
            if (data.keaAppsTotal && data.keaAppsTotal > 0) {
                this.menuItems[0].visible = true
                this.menuItems[1].items[0]['visible'] = true
            } else {
                this.menuItems[0].visible = false
                this.menuItems[1].items[0]['visible'] = false
            }
            // if there are BIND 9 apps then show BIND 9 related menu items
            // otherwise hide them
            if (data.bind9AppsTotal && data.bind9AppsTotal > 0) {
                this.menuItems[1].items[1]['visible'] = true
            } else {
                this.menuItems[1].items[1]['visible'] = false
            }
        })

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
                        label: 'Hosts',
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
                        label: 'Kea DHCP',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/kea/all',
                    },
                    {
                        label: 'BIND 9 DNS',
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

    ngOnInit() {
        this.generalApi.getVersion().subscribe((data) => {
            this.storkVersion = data.version
            this.storkBuildDate = data.date
        })

        // If Grafana url is not empty, we need to make
        // Services.Grafana menu choice visible and set it's url.
        // Otherwise we need to make sure it's not visible.
        this.settingSvc.getSettings().subscribe((data) => {
            const grafanaUrl = data['grafana_url']

            for (const menuItem of this.menuItems) {
                if (menuItem['label'] === 'Services') {
                    for (const subMenu of menuItem.items) {
                        if (subMenu['label'] === 'Grafana') {
                            if (grafanaUrl && grafanaUrl !== '') {
                                subMenu['visible'] = true
                                subMenu['url'] = grafanaUrl
                            } else {
                                subMenu['visible'] = false
                            }
                        }
                    }
                }
            }
        })
    }

    signOut() {
        this.router.navigate(['/logout'])
    }
}
