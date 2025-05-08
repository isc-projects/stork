import { AfterViewInit, Component, OnDestroy, OnInit } from '@angular/core'
import { Subscription } from 'rxjs'
import { MenuItem, MessageService } from 'primeng/api'
import { DHCPService, MigrationStatus } from '../backend'
import { ActivatedRoute } from '@angular/router'
import { getErrorMessage } from '../utils'

/**
 * This component implements a page which displays config migrations.
 * The list of hosts is paged.
 *
 * This component is also responsible for viewing given config migration
 * details in tab view, switching between tabs, closing them etc.
 */
@Component({
    selector: 'app-config-migration-page',
    templateUrl: './config-migration-page.component.html',
    styleUrl: './config-migration-page.component.sass',
})
export class ConfigMigrationPageComponent implements OnInit, OnDestroy, AfterViewInit {
    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     */
    subscriptions = new Subscription()

    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Config Migrations' }]

    /**
     * Array of tabs with migration information.
     *
     * The first tab is always present and displays the list.
     */
    tabs: MenuItem[]

    /**
     * Holds the information about specific config migrations presented in
     * the tabs.
     *
     * The tab holding hosts list is not included in this tab. If only a tab
     * with the hosts list is displayed, this array is empty.
     */
    tabItems: MigrationStatus[]

    /**
     * Selected tab index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Constructor.
     *
     * @param route activated route used to gather parameters from the URL.
     * @param dhcpApi server API used to gather hosts information.
     * @param messageService message service used to display error messages to a user.
     */
    constructor(
        private route: ActivatedRoute,
        private dhcpApi: DHCPService,
        private messageService: MessageService
    ) {}

    /**
     * Unsubscribe all subscriptions.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     *
     * It configures initial state of PrimeNG Menu tabs.
     */
    ngOnInit() {
        // Initially, there is only a tab with hosts list.
        this.tabs = [{ label: 'Config migrations', routerLink: '/config-migrations/all' }]
        this.tabItems = [{}]
    }

    /**
     * Component lifecycle hook called after Angular completed the initialization of the
     * component's view.
     *
     * We subscribe to router events to act upon URL and/or queryParams changes.
     * This is done at this step, because we have to be sure that all child components,
     * especially PrimeNG table in HostsTableComponent, are initialized.
     */
    ngAfterViewInit(): void {
        this.subscriptions.add(
            this.route.paramMap.subscribe((params) => {
                if (params.has('id')) {
                    const id = params.get('id')
                    if (id === 'all') {
                        this.switchToTab(0)
                    } else {
                        const idNumber = Number.parseInt(id, 10)
                        this.openTab(idNumber)
                    }
                }
            })
        )
    }

    /**
     * Opens existing or new tab.
     *
     * If the tab for the given ID does not exist, a new tab is opened.
     * Otherwise, the existing tab is opened.
     *
     * @param id config migration ID.
     */
    private openTab(id: number) {
        const index = this.tabs.findIndex((t, i) => this.tabItems[i].id === id)
        if (index >= 0) {
            this.switchToTab(index)
            return
        }

        // Make an API call to get migration details.
        this.dhcpApi.getMigration(id).subscribe({
            next: (status) => {
                this.createTab(status)
                this.switchToTab(this.tabs.length - 1)
            },
            error: (error) => {
                const errorMessage = getErrorMessage(error)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to get migration details',
                    detail: errorMessage,
                })
            },
        })
    }

    /**
     * Closes a tab.
     *
     * This function is called when user closes a selected tab. If the
     * user closed a currently selected tab, a previous tab becomes selected.
     *
     * @param event event generated when the tab is closed.
     * @param tabIndex index of the tab to be closed. It must be equal to or
     *        greater than 1.
     */
    closeTab(tabIndex: number, event?: Event) {
        if (event) {
            event.preventDefault()
            event.stopPropagation()
        }

        if (tabIndex === 0) {
            return
        }

        // Remove the MenuItem representing the tab.
        this.tabs = [...this.tabs.slice(0, tabIndex), ...this.tabs.slice(tabIndex + 1)]
        // Remove host specific information associated with the tab.
        this.tabItems = [...this.tabItems.slice(0, tabIndex), ...this.tabItems.slice(tabIndex + 1)]

        if (this.activeTabIndex === tabIndex) {
            // Closing currently selected tab. Switch to previous tab.
            this.switchToTab(tabIndex - 1)
        } else if (this.activeTabIndex > tabIndex) {
            // Sitting on the later tab then the one closed. We don't need
            // to switch, but we have to adjust the active tab index.
            this.activeTabIndex--
        }
    }

    /**
     * Selects an existing tab.
     *
     * @param tabIndex index of the tab to be selected.
     */
    private switchToTab(tabIndex: number) {
        if (this.activeTabIndex === tabIndex) {
            return
        }
        this.activeTabIndex = tabIndex
    }

    /**
     * Adds a new tab.
     *
     * @param status migration status to be added to the tab.
     */
    private createTab(status: MigrationStatus) {
        const routerLink = `/config-migrations/${status.id}`

        this.tabs = [
            ...this.tabs,
            {
                label: `Migration ${status.id}`,
                routerLink: routerLink,
            },
        ]

        this.tabItems = [...this.tabItems, status]
    }

    /**
     * Replaces the migration status in the tab.
     */
    private replaceItem(status: MigrationStatus) {
        const index = this.tabs.findIndex((t, i) => this.tabItems[i].id === status.id)
        if (index <= 0) {
            return
        }
        this.tabItems[index] = status
    }

    /**
     * Function called when requested to cancel a migration.
     */
    onCancelMigration(id: number) {
        const index = this.tabs.findIndex((t, i) => this.tabItems[i].id === id)
        if (index <= 0) {
            return
        }

        // ToDo: Make an API call to cancel migration.
        this.dhcpApi.putMigration(id).subscribe({
            next: (status) => {
                this.replaceItem(status)
            },
            error: (error) => {
                const errorMessage = getErrorMessage(error)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to cancel migration',
                    detail: errorMessage,
                })
            },
        })
    }

    /**
     * Function called when requested to clean up finished migrations.
     */
    onClearFinishedMigrations() {
        // Make an API call to clean up finished migrations.
        this.dhcpApi.deleteFinishedMigrations().subscribe({
            next: () => {
                // Close tabs for finished migrations.
                for (let i = this.tabs.length - 1; i > 0; i--) {
                    const item = this.tabItems[i]
                    if (item.endDate == null) {
                        continue
                    }
                    this.closeTab(i)
                }
            },
            error: (error) => {
                const errorMessage = getErrorMessage(error)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to clean up finished migrations',
                    detail: errorMessage,
                })
            },
        })
    }

    /**
     * Function called when requested to refresh migration status.
     */
    onRefreshMigration(id: number) {
        this.dhcpApi.getMigration(id).subscribe({
            next: (status) => {
                this.replaceItem(status)
            },
            error: (error) => {
                const errorMessage = getErrorMessage(error)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to refresh migration status',
                    detail: errorMessage,
                })
            },
        })
    }
}
