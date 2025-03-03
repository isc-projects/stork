import { Component, EventEmitter, OnInit, Output, ViewChild } from '@angular/core'
import { LazyLoadTable } from '../table'
import { DHCPService, MigrationStatus } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { lastValueFrom } from 'rxjs'

/**
 * This component implements a table of configuration migrations.
 * The list of migrations is paged.
 */
@Component({
    selector: 'app-config-migration-table',
    templateUrl: './config-migration-table.component.html',
    styleUrl: './config-migration-table.component.sass',
})
export class ConfigMigrationTableComponent extends LazyLoadTable<MigrationStatus> implements OnInit {
    /**
     * Event emitted when the user wants to clear finished migrations.
     */
    @Output() clear = new EventEmitter<void>()

    /**
     * Event emitted when the user wants to cancel a migration.
     * Emits the ID of the migration to cancel.
     */
    @Output() cancel = new EventEmitter<string>()

    /**
     * PrimeNG table instance.
     */
    @ViewChild('configMigrationTable') table: Table

    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService
    ) {
        super()
    }

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

        lastValueFrom(this.dhcpApi.getMigrations(event.first, event.rows))
            .then((data) => {
                this.migrations = data.items ?? []
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
     * Lifecycle hook invoked when the component is initialized.
     *
     * Load the initial data.
     */
    ngOnInit() {
        this.loadData({ first: 0, rows: 10 })
    }

    /**
     * Returns all currently displayed configuration migrations.
     */
    get migrations(): MigrationStatus[] {
        return this.dataCollection
    }

    /**
     * Sets configuration migrations to be displayed.
     */
    set migrations(migrations: MigrationStatus[]) {
        this.dataCollection = migrations
    }

    /**
     * Emits an event to clear finished migrations
     */
    clearFinishedMigrations() {
        this.clear.emit()
    }

    /**
     * Emits an event to cancel a specific migration
     *
     * @param migrationId ID of the migration to cancel
     */
    cancelMigration(migrationId: string) {
        this.cancel.emit(migrationId)
    }
}
