import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild } from '@angular/core'
import { LazyLoadTable } from '../table'
import { DHCPService, MigrationStatus } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ConfirmationService, MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { lastValueFrom, Observable, Subscription } from 'rxjs'

/**
 * This component implements a table of configuration migrations.
 * The list of migrations is paged.
 */
@Component({
    selector: 'app-config-migration-table',
    templateUrl: './config-migration-table.component.html',
    styleUrl: './config-migration-table.component.sass',
})
export class ConfigMigrationTableComponent extends LazyLoadTable<MigrationStatus> implements OnInit, OnDestroy {
    /**
     * Keeps all the RxJS subscriptions.
     */
    subscriptions: Subscription = new Subscription()

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
     * Spawn of migration statuses that are being updated externally.
     * The null value indicates that unknown number of statuses were updated.
     */
    @Input() alteredStatuses: Observable<MigrationStatus> = new Observable<MigrationStatus>()

    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private confirmationService: ConfirmationService
    ) {
        super()
        this.dataLoading = true
    }

    /**
     * Registers for migration statuses updates.
     */
    ngOnInit() {
        // Register for migration statuses updates.
        this.subscriptions.add(
            this.alteredStatuses.subscribe((status) => {
                if (status && this.dataCollection) {
                    const index = this.dataCollection.findIndex((s) => s.id === status.id)
                    if (index !== -1) {
                        this.dataCollection = [
                            ...this.dataCollection.slice(0, index),
                            status,
                            ...this.dataCollection.slice(index + 1),
                        ]
                    }
                } else {
                    this.loadData({ first: 0, rows: this.table.rows })
                }
            })
        )
    }

    /**
     * Does a cleanup when the component is destroyed.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
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
