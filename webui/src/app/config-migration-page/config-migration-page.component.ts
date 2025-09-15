import { Component, signal, viewChild } from '@angular/core'
import { lastValueFrom } from 'rxjs'
import { MessageService } from 'primeng/api'
import { DHCPService, MigrationStatus } from '../backend'
import { getErrorMessage } from '../utils'
import { ConfigMigrationTableComponent } from '../config-migration-table/config-migration-table.component'
import { TabViewComponent } from '../tab-view/tab-view.component'

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
export class ConfigMigrationPageComponent {
    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Config Migrations' }]

    migrationProvider: (id: number) => Promise<MigrationStatus> = (id) => lastValueFrom(this.dhcpApi.getMigration(id))
    tabTitleProvider: (entity: MigrationStatus) => string = (entity: MigrationStatus) => `Migration ${entity.id}`

    migrationsTableComponent = viewChild(ConfigMigrationTableComponent)

    tabView = viewChild(TabViewComponent)

    activeTabMigrationID = signal<number>(undefined)

    /**
     * Constructor.
     *
     * @param dhcpApi server API used to gather hosts information.
     * @param messageService message service used to display error messages to a user.
     */
    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService
    ) {}

    /**
     * Function called when requested to cancel a migration.
     */
    onCancelMigration(id: number) {
        // ToDo: Make an API call to cancel migration.
        this.dhcpApi.putMigration(id).subscribe({
            next: (status) => {
                this.tabView()?.onUpdateTabEntity(id, status)
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
                this.tabView()?.closeTabsConditionally((migration: MigrationStatus) => migration.endDate != null)

                // We don't know which migration statuses were deleted. Reload table data.
                this.migrationsTableComponent().table.clear()
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
}
