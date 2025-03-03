import { Component, EventEmitter, Input, Output } from '@angular/core'
import { MigrationError, MigrationStatus } from '../backend'

/**
 * Component presenting details for a selected configuration migration.
 */
@Component({
    selector: 'app-config-migration-tab',
    templateUrl: './config-migration-tab.component.html',
    styleUrl: './config-migration-tab.component.sass',
})
export class ConfigMigrationTabComponent {
    /**
     * An event emitter notifying a parent that user has clicked the
     * Cancel button to cancel the migration.
     */
    @Output() cancel = new EventEmitter<number>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Refresh button to refresh migration status.
     */
    @Output() refresh = new EventEmitter<number>()

    /**
     * Structure containing migration information to be displayed.
     */
    @Input() migration: MigrationStatus

    /**
     * Determines if the migration is currently running.
     */
    get isRunning(): boolean {
        return !this.migration.endDate && !this.migration.canceling
    }

    /**
     * Determines if the migration is currently canceling.
     */
    get isCanceling(): boolean {
        return !this.migration.endDate && this.migration.canceling
    }

    /**
     * Determines if the migration has completed successfully.
     */
    get isCompleted(): boolean {
        return !!this.migration.endDate && !this.migration.generalError
    }

    /**
     * Determines if the migration has failed.
     */
    get isFailed(): boolean {
        return !!this.migration.endDate && !!this.migration.generalError
    }

    /**
     * Gets the completion percentage of the migration.
     */
    get completionPercentage(): number {
        return (this.migration.processedItemsCount / this.migration.totalItemsCount) * 100
    }

    /**
     * Gets the total number of errors in the migration.
     */
    get totalErrors(): number {
        return this.migration.errors?.total || 0
    }

    /**
     * Gets the error items from the migration.
     */
    get errorItems(): MigrationError[] {
        return this.migration.errors?.items || []
    }

    /**
     * Gets the status of the migration as a text string.
     */
    get statusText(): string {
        if (this.isRunning) return 'Running'
        if (this.isCanceling) return 'Canceling'
        if (this.isCompleted) return 'Completed'
        if (this.isFailed) return 'Failed'
        return 'Unknown'
    }

    /**
     * Gets the severity level for the status tag.
     */
    get statusSeverity(): string {
        if (this.isRunning) return 'info'
        if (this.isCanceling) return 'warning'
        if (this.isCompleted) return 'success'
        if (this.isFailed) return 'danger'
        return 'secondary'
    }

    /**
     * Emits an event to refresh migration status.
     */
    refreshMigration() {
        this.refresh.emit(this.migration.id)
    }

    /**
     * Emits an event to cancel the migration.
     */
    cancelMigration() {
        this.cancel.emit(this.migration.id)
    }
}
