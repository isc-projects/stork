import { Component, EventEmitter, Input, Output } from '@angular/core'
import { MigrationError, MigrationStatus } from '../backend'
import { AuthService } from '../auth.service'
import { NgIf, NgSwitch, NgSwitchCase, NgSwitchDefault } from '@angular/common'
import { Fieldset } from 'primeng/fieldset'
import { Tag } from 'primeng/tag'
import { ProgressBar } from 'primeng/progressbar'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { TableModule } from 'primeng/table'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { DurationPipe } from '../pipes/duration.pipe'

/**
 * Component presenting details for a selected configuration migration.
 */
@Component({
    selector: 'app-config-migration-tab',
    templateUrl: './config-migration-tab.component.html',
    styleUrl: './config-migration-tab.component.sass',
    imports: [
        NgIf,
        Fieldset,
        Tag,
        ProgressBar,
        EntityLinkComponent,
        Button,
        ManagedAccessDirective,
        TableModule,
        NgSwitch,
        NgSwitchCase,
        NgSwitchDefault,
        LocaltimePipe,
        DurationPipe,
    ],
})
export class ConfigMigrationTabComponent {
    /**
     * An event emitter notifying a parent that user has clicked the
     * Cancel button to cancel the migration.
     */
    @Output() cancelMigration = new EventEmitter<number>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Refresh button to refresh migration status.
     */
    @Output() refreshMigration = new EventEmitter<number>()

    /**
     * Structure containing migration information to be displayed.
     */
    @Input({ required: true }) migration: MigrationStatus

    /**
     * Boolean flag stating whether user has privileges to cancel an ongoing migration.
     */
    get canCancelMigration(): boolean {
        return this.authService.hasPrivilege('migrations', 'update')
    }

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
        if (this.migration.totalItemsCount === 0) {
            return 0
        }
        return Math.round((this.migration.processedItemsCount / this.migration.totalItemsCount) * 100)
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
    get statusSeverity(): 'info' | 'warn' | 'success' | 'danger' | 'secondary' {
        if (this.isRunning) return 'info'
        if (this.isCanceling) return 'warn'
        if (this.isCompleted) return 'success'
        if (this.isFailed) return 'danger'
        return 'secondary'
    }

    /**
     * Emits an event to refresh migration status.
     */
    onRefresh() {
        this.refreshMigration.emit(this.migration.id)
    }

    /**
     * Emits an event to cancel the migration.
     */
    onCancel() {
        this.cancelMigration.emit(this.migration.id)
    }

    /**
     * Component class constructor.
     * @param authService Auth service used to check user authorization for canceling migrations.
     */
    constructor(private authService: AuthService) {}
}
