import { ComponentFixture, TestBed } from '@angular/core/testing'
import { ConfigMigrationTabComponent } from './config-migration-tab.component'
import { MigrationStatus } from '../backend'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { TableModule } from 'primeng/table'
import { FieldsetModule } from 'primeng/fieldset'
import { TagModule } from 'primeng/tag'
import { ProgressBarModule } from 'primeng/progressbar'

describe('ConfigMigrationTabComponent', () => {
    let component: ConfigMigrationTabComponent
    let fixture: ComponentFixture<ConfigMigrationTabComponent>

    const mockRunningMigration: Partial<MigrationStatus> = {
        id: 1,
        startDate: new Date(2023, 5, 15, 10, 0, 0).toISOString(),
        endDate: null,
        canceling: false,
        processedItemsCount: 5,
        totalItemsCount: 10,
        errors: {
            total: 0,
            items: [],
        },
        elapsedTime: '10m',
        estimatedLeftTime: '5m',
    }

    const mockCompletedMigration: Partial<MigrationStatus> = {
        id: 2,
        startDate: new Date(2023, 5, 15, 10, 0, 0).toISOString(),
        endDate: new Date(2023, 5, 15, 10, 15, 0).toISOString(),
        canceling: false,
        processedItemsCount: 10,
        totalItemsCount: 10,
        errors: {
            total: 0,
            items: [],
        },
        elapsedTime: '15m',
        estimatedLeftTime: '0s',
    }

    const mockFailedMigration: Partial<MigrationStatus> = {
        id: 3,
        startDate: new Date(2023, 5, 15, 10, 0, 0).toISOString(),
        endDate: new Date(2023, 5, 15, 10, 8, 0).toISOString(),
        canceling: false,
        processedItemsCount: 3,
        totalItemsCount: 10,
        errors: {
            total: 2,
            items: [
                { id: 1, error: 'Failed to process host', label: 'host-1', causeEntity: 'host' },
                { id: 2, error: 'Failed to process host', label: 'host-2', causeEntity: 'host' },
            ],
        },
        generalError: 'Migration failed due to errors',
        elapsedTime: '8m',
        estimatedLeftTime: '0s',
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [NoopAnimationsModule, TableModule, FieldsetModule, TagModule, ProgressBarModule],
            declarations: [ConfigMigrationTabComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigMigrationTabComponent)
        component = fixture.componentInstance
        component.migration = mockRunningMigration as MigrationStatus
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should emit refresh event when refresh button is clicked', () => {
        spyOn(component.refresh, 'emit')

        const refreshButton = fixture.debugElement.query(By.css('button:not(.p-button-danger)'))
        refreshButton.nativeElement.click()

        expect(component.refresh.emit).toHaveBeenCalledWith(1)
    })

    it('should emit event when cancel button is clicked', () => {
        spyOn(component.cancel, 'emit')

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        cancelButton.nativeElement.click()

        expect(component.cancel.emit).toHaveBeenCalledWith(1)
    })

    it('should correctly identify migration state based on endDate', () => {
        // Test running migration
        component.migration = { ...mockRunningMigration } as MigrationStatus
        expect(component.isRunning).toBe(true)
        expect(component.isCanceling).toBe(false)
        expect(component.isCompleted).toBe(false)
        expect(component.isFailed).toBe(false)

        // Test canceling migration
        component.migration = { ...mockRunningMigration, canceling: true } as MigrationStatus
        expect(component.isRunning).toBe(false)
        expect(component.isCanceling).toBe(true)
        expect(component.isCompleted).toBe(false)
        expect(component.isFailed).toBe(false)

        // Test completed migration
        component.migration = mockCompletedMigration as MigrationStatus
        expect(component.isRunning).toBe(false)
        expect(component.isCanceling).toBe(false)
        expect(component.isCompleted).toBe(true)
        expect(component.isFailed).toBe(false)

        // Test failed migration
        component.migration = mockFailedMigration as MigrationStatus
        expect(component.isRunning).toBe(false)
        expect(component.isCanceling).toBe(false)
        expect(component.isCompleted).toBe(false)
        expect(component.isFailed).toBe(true)
    })
})
