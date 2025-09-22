import { Component, EventEmitter, Output, ViewChild } from '@angular/core'
import { DHCPService, MigrationStatus } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ConfirmationService, MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { lastValueFrom } from 'rxjs'

/**
 * This component implements a table of configuration migrations.
 * The list of migrations is paged.
 */
@Component({
    selector: 'app-config-migration-table',
    standalone: false,
    templateUrl: './config-migration-table.component.html',
    styleUrl: './config-migration-table.component.sass',
})
export class ConfigMigrationTableComponent {
    /**
     * Event emitted when the user wants to clear finished migrations.
     */
    @Output() clearMigrations = new EventEmitter<void>()

    /**
     * Event emitted when the user wants to cancel a migration.
     * Emits the ID of the migration to cancel.
     */
    @Output() cancelMigration = new EventEmitter<number>()

    /**
     * PrimeNG table instance.
     */
    @ViewChild('configMigrationTable') table: Table

    /**
     * Flag keeping track of whether table data is loading.
     */
    dataLoading: boolean

    /**
     * Data collection of migrations in the table.
     */
    dataCollection: MigrationStatus[] = []

    /**
     * Total number of records in the table.
     */
    totalRecords: number = 0

    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService
    ) {}

    /**
     * Loads configuration migrations from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned. If it is not specified, the current
     * values are used when available.
     */
    loadData(event: TableLazyLoadEvent) {
        // Indicate that migrations refresh is in progress.
        this.dataLoading = true

        lastValueFrom(this.dhcpApi.getMigrations(event?.first ?? this.table.first, event?.rows ?? this.table.rows))
            .then((data) => {
                this.dataCollection = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get configuration migrations list',
                    detail: 'Error getting configuration migrations list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    /**
     * Emits an event to clear finished migrations
     */
    clearFinishedMigrations() {
        this.confirmationService.confirm({
            message: 'Are you sure you want to clear finished migrations?',
            header: 'Clear finished migrations',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.clearMigrations.emit()
            },
        })
    }

    /**
     * Emits an event to cancel a specific migration
     *
     * @param migrationId ID of the migration to cancel
     */
    cancel(migrationId: number) {
        this.cancelMigration.emit(migrationId)
    }

    /**
     * Gets the completion percentage of the specific migration.
     */
    getCompletionPercentage(status: MigrationStatus): number {
        if (status.totalItemsCount === 0) {
            return 0
        }
        return Math.round((status.processedItemsCount / status.totalItemsCount) * 100)
    }
}
