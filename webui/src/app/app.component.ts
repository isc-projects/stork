import { Component, OnDestroy, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable, Subscription } from 'rxjs'

import { MenuItem } from 'primeng/api'

import { GeneralService } from './backend/api/api'
import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'
import { SettingService } from './setting.service'
import { ServerDataService } from './server-data.service'
import { Settings, User } from './backend'
import { ThemeService } from './theme.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit, OnDestroy {
    storkVersion = 'unknown'
    storkBuildDate = 'unknown'
    currentUser: User = null
    loadingInProgress = new Observable()
    userMenuItems: MenuItem[]

    menuItems: MenuItem[]

    breadcrumbItems: MenuItem[]

    /**
     * Holds information if dark theme is applied.
     */
    isDark: boolean

    /**
     * Keeps all the RxJS subscriptions.
     */
    subscriptions: Subscription = new Subscription()

    constructor(
        private router: Router,
        private serverData: ServerDataService,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService,
        private settingSvc: SettingService,
        private themeService: ThemeService
    ) {
        this.initMenus()

        this.breadcrumbItems = [{ label: 'Categories' }]

        this.loadingInProgress = this.loadingService.getState()
    }

    initMenus() {
        this.userMenuItems = [
            {
                label: 'Profile',
                id: 'profile',
                icon: 'fa fa-cog',
                routerLink: '/profile',
            },
        ]

        this.menuItems = [
            {
                label: 'DHCP',
                id: 'dhcp',
                visible: false,
                // badgeStyleClass: 'error',
                badge: '0',
                items: [
                    {
                        label: 'Dashboard',
                        id: 'dashboard',
                        icon: 'fa fa-tachometer-alt',
                        routerLink: '/dashboard',
                        badge: '1',
                    },
                    {
                        label: 'Leases Search',
                        id: 'leases-search',
                        icon: 'fa fa-search',
                        routerLink: '/dhcp/leases',
                    },
                    {
                        label: 'Host Reservations',
                        id: 'host-reservations',
                        icon: 'fa fa-laptop',
                        routerLink: '/dhcp/hosts',
                    },
                    {
                        label: 'Subnets',
                        id: 'subnets',
                        icon: 'fa fa-project-diagram',
                        routerLink: '/dhcp/subnets',
                    },
                    {
                        label: 'Shared Networks',
                        id: 'shared-networks',
                        icon: 'fa fa-network-wired',
                        routerLink: '/dhcp/shared-networks',
                    },
                ],
            },
            {
                label: 'Services',
                id: 'services',
                items: [
                    {
                        label: 'Kea Apps',
                        id: 'kea-apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/kea/all',
                    },
                    {
                        label: 'BIND 9 Apps',
                        id: 'bind9-apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/bind9/all',
                    },
                    {
                        label: 'Machines',
                        id: 'machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/authorized',
                    },
                    {
                        label: 'Grafana',
                        id: 'grafana',
                        icon: 'pi pi-chart-line',
                        url: '',
                        visible: false,
                    },
                ],
            },
            {
                label: 'Monitoring',
                id: 'monitoring',
                items: [
                    {
                        label: 'Events',
                        id: 'events',
                        icon: 'fa fa-calendar-times',
                        routerLink: '/events',
                    },
                    {
                        label: 'Communication',
                        id: 'communication',
                        icon: 'fa fa-signal',
                        routerLink: '/communication',
                    },
                    {
                        label: 'Software versions',
                        id: 'versions',
                        icon: 'pi pi-history',
                        routerLink: '/versions',
                    },
                ],
            },
            {
                label: 'Configuration',
                id: 'configuration',
                items: [
                    {
                        label: 'Users',
                        id: 'users',
                        visible: false,
                        icon: 'fa fa-user',
                        routerLink: '/users',
                    },
                    {
                        label: 'Review Checkers',
                        id: 'checkers',
                        icon: 'fa fa-tasks',
                        routerLink: '/review-checkers',
                    },
                    {
                        label: 'Settings',
                        id: 'settings',
                        icon: 'fa fa-cog',
                        routerLink: '/settings',
                    },
                ],
            },
            {
                label: 'Help',
                id: 'help',
                items: [
                    {
                        label: 'Stork Manual',
                        id: 'stork-manual',
                        icon: 'fa fa-book',
                        routerLink: '/assets/arm/index.html',
                        target: 'blank',
                    },
                    {
                        label: 'Stork API Docs (SwaggerUI)',
                        id: 'stork-API-docs-swagger',
                        icon: 'fa fa-code',
                        routerLink: '/swagger-ui',
                    },
                    {
                        label: 'Stork API Docs (Redoc)',
                        id: 'stork-API-docs-redoc',
                        icon: 'fa fa-code',
                        routerLink: '/api/docs',
                        target: 'blank',
                    },
                    {
                        label: 'BIND 9 Manual',
                        id: 'bind9-manual',
                        icon: 'fa fa-book',
                        url: 'https://bind9.readthedocs.io/',
                        target: 'blank',
                    },
                    {
                        label: 'Kea Manual',
                        id: 'kea-manual',
                        icon: 'fa fa-book',
                        url: 'https://kea.readthedocs.io/',
                        target: 'blank',
                    },
                ],
            },
        ]
    }

    /**
     * Get menu item or subitem from Stork menu based on provided name.
     *
     * @param name A menu item name that must exist in Stork menu tree
     *             that is defined in this.menuItems.
     * @returns A reference to found menu item or null if not found.
     */
    getMenuItem(name) {
        for (const menuItem of this.menuItems) {
            if (menuItem['label'] === name) {
                return menuItem
            }
            for (const subMenu of menuItem.items) {
                if (subMenu['label'] === name) {
                    return subMenu
                }
            }
        }
        console.error('menu item not found', name)
        return null
    }

    ngOnInit() {
        this.subscriptions.add(
            this.generalApi.getVersion().subscribe((data) => {
                this.storkVersion = data.version
                this.storkBuildDate = data.date
            })
        )

        this.subscriptions.add(
            this.auth.currentUser.subscribe((x) => {
                this.currentUser = x
                const menuItem = this.getMenuItem('Users')
                if (this.auth.superAdmin()) {
                    // super admin can see Configuration/Users menu
                    menuItem['visible'] = true
                } else {
                    menuItem['visible'] = false
                }
                // force refresh of top menu in UI
                this.menuItems = [...this.menuItems]

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

                        // force refresh of top menu in UI
                        this.menuItems = [...this.menuItems]
                    })

                    // If Grafana url is not empty, we need to make
                    // Services.Grafana menu choice visible and set it's url.
                    // Otherwise we need to make sure it's not visible.
                    this.settingSvc.getSettings().subscribe((data: Settings) => {
                        const grafanaUrl = data?.grafanaUrl

                        const grafanaMenuItem = this.getMenuItem('Grafana')

                        if (grafanaUrl && grafanaUrl !== '') {
                            grafanaMenuItem['visible'] = true
                            grafanaMenuItem['url'] = grafanaUrl
                        } else {
                            grafanaMenuItem['visible'] = false
                        }

                        // force refresh of top menu in UI
                        this.menuItems = [...this.menuItems]
                    })
                }
            })
        )

        // Subscribe to themeService isDark$ BehaviorSubject observable,
        // to get notified of dark/light mode change.
        this.subscriptions.add(
            this.themeService.isDark$.subscribe((isDark) => {
                this.isDark = isDark
            })
        )
        // Sets initial dark/light mode theme basing on user's preference and OS/browser settings.
        this.setInitialTheme()
    }

    /**
     * Does a cleanup when the component is destroyed.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    signOut() {
        this.router.navigate(['/logout'])
    }

    /**
     * Communicates with themeService to change stork UI dark/light mode and the color theme.
     * User's preference is stored in browser's local storage.
     * @param isDark when true provided, dark mode is enabled; otherwise light mode is enabled
     * @param theme optional color theme; when skipped, default theme and color scheme is used
     */
    changeTheme(isDark: boolean = false, theme?: string): void {
        this.themeService.switchTheme(isDark, theme)
        this.themeService.storeTheme()
    }

    /**
     * Communicates with themeService to set the initial dark/light mode theme
     * basing on user's preference and OS/browser settings.
     */
    setInitialTheme(): void {
        this.themeService.setInitialTheme()
    }
}
