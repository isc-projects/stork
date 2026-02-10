import { Component, OnDestroy, OnInit } from '@angular/core'
import { Router, RouterLink, RouterOutlet } from '@angular/router'
import { firstValueFrom, Observable, Subscription } from 'rxjs'

import { MenuItem, MessageService } from 'primeng/api'

import { GeneralService } from './backend/api/api'
import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'
import { SettingService } from './setting.service'
import { ServerDataService } from './server-data.service'
import { Settings, User } from './backend'
import { ThemeService } from './theme.service'
import { Severity, VersionService } from './version.service'
import { NgIf, AsyncPipe } from '@angular/common'
import { Menubar } from 'primeng/menubar'
import { Tooltip } from 'primeng/tooltip'
import { Ripple } from 'primeng/ripple'
import { ToggleButton } from 'primeng/togglebutton'
import { FormsModule } from '@angular/forms'
import { ProgressSpinner } from 'primeng/progressspinner'
import { GlobalSearchComponent } from './global-search/global-search.component'
import { SplitButton } from 'primeng/splitbutton'
import { PriorityErrorsPanelComponent } from './priority-errors-panel/priority-errors-panel.component'
import { Toast } from 'primeng/toast'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
    imports: [
        NgIf,
        Menubar,
        RouterLink,
        Tooltip,
        Ripple,
        ToggleButton,
        FormsModule,
        ProgressSpinner,
        GlobalSearchComponent,
        SplitButton,
        PriorityErrorsPanelComponent,
        Toast,
        RouterOutlet,
        AsyncPipe,
    ],
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

    /**
     * Flag stating whether to display badge for some menu items to drag user's attention about ISC software versions.
     * @private
     */
    private displaySwVersionBadge: boolean = false

    /**
     * Severity for the badge for some menu items to drag user's attention about ISC software versions.
     * @private
     */
    private swVersionBadgeSeverity: Severity

    /**
     * Custom style of the toggle button.
     */
    darkModeToggleButton = {
        root: {
            contentSmPadding: '0.25rem',
        },
    }

    /**
     * Top menubar style.
     */
    appMenubar = {
        root: {
            padding: '0',
            borderRadius: '0',
            background: '{primary.600}',
            borderColor: '{primary.600}',
        },
        colorScheme: {
            light: {
                root: {
                    mobileButtonColor: '{primary.contrast.color}',
                },
            },
        },
    }

    constructor(
        private router: Router,
        private serverData: ServerDataService,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService,
        private settingSvc: SettingService,
        private themeService: ThemeService,
        private versionService: VersionService,
        private messageService: MessageService
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
                items: [
                    {
                        label: 'Dashboard',
                        id: 'dashboard',
                        icon: 'fa fa-tachometer-alt',
                        routerLink: '/dashboard',
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
                    {
                        label: 'Config migrations',
                        id: 'config-migrations',
                        routerLink: '/config-migrations',
                        icon: 'fa fa-suitcase',
                    },
                ],
            },
            {
                label: 'DNS',
                id: 'dns',
                visible: false,
                items: [
                    {
                        label: 'Dashboard',
                        id: 'dns-dashboard',
                        icon: 'fa fa-tachometer-alt',
                        routerLink: '/dashboard',
                    },
                    { label: 'Zones', id: 'zones', icon: 'pi pi-sitemap', routerLink: '/dns/zones' },
                ],
            },
            {
                label: 'Services',
                id: 'services',
                items: [
                    {
                        label: 'All Daemons',
                        id: 'all-daemons',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/daemons/all',
                    },
                    {
                        label: 'Kea Daemons',
                        id: 'kea-daemons',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/daemons/all',
                        queryParams: { daemons: ['dhcp4', 'dhcp6', 'd2', 'ca', 'netconf'] },
                    },
                    {
                        label: 'DNS Daemons',
                        id: 'dns-daemons',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/daemons/all',
                        queryParams: { daemons: ['named', 'pdns'] },
                    },
                    {
                        label: 'Machines',
                        id: 'machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/all',
                        queryParams: { authorized: 'true' },
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
        this.messageService.add({
            severity: 'error',
            summary: 'Error getting menu item',
            detail: 'Menu item not found: ' + name,
            life: 10000,
        })
        return null
    }

    ngOnInit() {
        this.subscriptions.add(
            this.generalApi.getVersion().subscribe((data) => {
                this.storkVersion = data.version
                this.storkBuildDate = data.date
                this.versionService.setStorkServerVersion(data.version)
            })
        )

        this.subscriptions.add(
            this.versionService.getVersionAlert().subscribe((data) => {
                this.displaySwVersionBadge = data.detected
                this.swVersionBadgeSeverity = data.severity
                this.setMenuItemBadges()
            })
        )

        this.subscriptions.add(
            this.auth.currentUser$.subscribe((x) => {
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
                    // Use firstValueFrom to subscribe to the observable and unsubscribe as soon as first value arrives.
                    // This is to check Stork server updates and unsubscribe.
                    firstValueFrom(this.versionService.getCurrentData()).catch((err) =>
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Error retrieving software versions data',
                            detail: 'Error occurred when retrieving software versions data: ' + err,
                            life: 10000,
                        })
                    )

                    this.serverData.getDaemonsStats().subscribe((data) => {
                        // if there are Kea daemons then show DHCP/Kea menu items
                        // otherwise hide them
                        const dhcpMenuItem = this.getMenuItem('DHCP')
                        const keaDaemonsMenuItem = this.getMenuItem('Kea Daemons')
                        if (data.dhcpDaemonsTotal) {
                            dhcpMenuItem.visible = true
                            keaDaemonsMenuItem['visible'] = true
                        } else {
                            dhcpMenuItem.visible = false
                            keaDaemonsMenuItem['visible'] = false
                        }
                        // if there are DNS daemons then show DNS related menu items
                        // otherwise hide them
                        const dnsDaemonsMenuItem = this.getMenuItem('DNS Daemons')
                        const dnsMenuItem = this.getMenuItem('DNS')
                        if (data.dnsDaemonsTotal) {
                            dnsDaemonsMenuItem['visible'] = true
                            dnsMenuItem['visible'] = true
                        } else {
                            dnsDaemonsMenuItem['visible'] = false
                            dnsMenuItem['visible'] = false
                        }
                        // if there are any of DHCP or DNS daemons, show related menu item; otherwise hide it
                        const allDaemonsMenuItem = this.getMenuItem('All Daemons')
                        allDaemonsMenuItem['visible'] = !!(data.dnsDaemonsTotal || data.dhcpDaemonsTotal)

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
     * Communicates with themeService to change stork UI dark/light mode.
     * User's preference is stored in browser's local storage.
     * @param isDark when true provided, dark mode is enabled; otherwise light mode is enabled
     */
    changeTheme(isDark: boolean = false): void {
        this.themeService.switchTheme(isDark)
        this.themeService.storeTheme()
    }

    /**
     * Communicates with themeService to set the initial dark/light mode theme
     * basing on user's preference and OS/browser settings.
     */
    setInitialTheme(): void {
        this.themeService.setInitialTheme()
    }

    /**
     * Returns appropriate class to style the badge basing on the severity.
     */
    get swVersionBadgeClass() {
        return this.swVersionBadgeSeverity === Severity.error ? 'p-badge-danger' : 'p-badge-warn'
    }

    /**
     * Enables or disable badges for some top menubar menu items and styles them appropriately.
     */
    setMenuItemBadges() {
        if (this.displaySwVersionBadge) {
            let item = this.getMenuItem('Software versions')
            item.badge = ' '
            item.badgeStyleClass = 'p-badge p-badge-dot ml-1 mb-2 ' + this.swVersionBadgeClass

            item = this.getMenuItem('Monitoring')
            item.badge = ' '
            item.badgeStyleClass = 'p-badge p-badge-dot ml-1 mb-2 ' + this.swVersionBadgeClass
        } else {
            let i = this.getMenuItem('Software versions')
            i.badge = undefined
            i = this.getMenuItem('Monitoring')
            i.badge = undefined
        }

        this.menuItems = [...this.menuItems]
    }
}
