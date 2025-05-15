import { ComponentFixture, TestBed } from '@angular/core/testing'
import { ConfigMigrationTabComponent } from './config-migration-tab.component'
import { MigrationStatus } from '../backend'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { TableModule } from 'primeng/table'
import { FieldsetModule } from 'primeng/fieldset'
import { TagModule } from 'primeng/tag'
import { ProgressBarModule } from 'primeng/progressbar'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { Table } from 'primeng/table'
import { ProgressBar } from 'primeng/progressbar'
import { Tag } from 'primeng/tag'
import { Fieldset } from 'primeng/fieldset'
import { ButtonModule } from 'primeng/button'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { provideRouter, RouterModule } from '@angular/router'

describe('ConfigMigrationTabComponent', () => {
    let component: ConfigMigrationTabComponent
    let fixture: ComponentFixture<ConfigMigrationTabComponent>

    const mockRunningMigration: Partial<MigrationStatus> = {
        id: 1,
        startDate: new Date(2023, 5, 15, 10, 0, 0).toISOString(),
        endDate: null,
        canceling: false,
        processedItemsCount: 0,
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
            imports: [
                NoopAnimationsModule,
                TableModule,
                FieldsetModule,
                TagModule,
                ProgressBarModule,
                ButtonModule,
                RouterModule,
            ],
            declarations: [ConfigMigrationTabComponent, EntityLinkComponent, LocaltimePipe, DurationPipe],
            providers: [provideRouter([])],
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
        spyOn(component.refreshMigration, 'emit')

        const refreshButton = fixture.debugElement.query(By.css('button:not(.p-button-danger)'))
        refreshButton.nativeElement.click()

        expect(component.refreshMigration.emit).toHaveBeenCalledWith(1)
    })

    it('should emit event when cancel button is clicked', () => {
        spyOn(component.cancelMigration, 'emit')

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        cancelButton.nativeElement.click()

        expect(component.cancelMigration.emit).toHaveBeenCalledWith(1)
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

    it('should display completion percentage', () => {
        component.migration = {
            ...mockRunningMigration,
            processedItemsCount: 6,
            totalItemsCount: 10,
        } as MigrationStatus
        fixture.detectChanges()

        const progressBarElement = fixture.debugElement.query(By.directive(ProgressBar))
        expect(progressBarElement).toBeTruthy()
        const progressBar = progressBarElement.componentInstance as ProgressBar
        expect(progressBar.value).toBe(60)
    })

    it('should display 100% completion for finished migration', () => {
        component.migration = {
            ...mockCompletedMigration,
            processedItemsCount: 10,
            totalItemsCount: 10,
        } as MigrationStatus
        fixture.detectChanges()

        const progressBarElement = fixture.debugElement.query(By.directive(ProgressBar))
        expect(progressBarElement).toBeTruthy()
        const progressBar = progressBarElement.componentInstance as ProgressBar
        expect(progressBar.value).toBe(100)
    })

    it('should display error items in table', () => {
        component.migration = mockFailedMigration as MigrationStatus
        fixture.detectChanges()

        const tableElement = fixture.debugElement.query(By.directive(Table))
        expect(tableElement).toBeTruthy()
        const table = tableElement.componentInstance as Table
        expect(table.value.length).toBe(2)

        // Get the table rows through the component instance
        const rows = table.value
        expect(rows[0].label).toBe('host-1')
        expect(rows[0].error).toBe('Failed to process host')
        expect(rows[1].label).toBe('host-2')
        expect(rows[1].error).toBe('Failed to process host')
    })

    it('should show time information appropriately', () => {
        // For running migration
        component.migration = mockRunningMigration as MigrationStatus
        fixture.detectChanges()
        let content = fixture.debugElement.nativeElement.textContent
        expect(content).toContain('Duration:10 minutes')
        expect(content).toContain('Estimated Left:5 minutes')

        // For completed migration
        component.migration = mockCompletedMigration as MigrationStatus
        fixture.detectChanges()
        content = fixture.debugElement.nativeElement.textContent
        expect(content).toContain('Duration:15 minutes')
        expect(content).not.toContain('Estimated Left')
    })

    it('should display general error in fieldset when present', () => {
        component.migration = mockFailedMigration as MigrationStatus
        fixture.detectChanges()

        const fieldsetElements = fixture.debugElement.queryAll(By.directive(Fieldset))
        expect(fieldsetElements.length).toBe(2)

        // First fieldset should be for status
        const statusFieldset = fieldsetElements[0].componentInstance as Fieldset
        expect(statusFieldset.legend).toBe('Status')

        // Second fieldset should be for errors
        const errorFieldset = fieldsetElements[1].componentInstance as Fieldset
        expect(errorFieldset.legend).toBe('Errors')
    })

    it('should disable cancel button for completed migration', () => {
        component.migration = mockCompletedMigration as MigrationStatus
        fixture.detectChanges()

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        expect(cancelButton.nativeElement.disabled).toBeTrue()
    })

    it('should disable cancel button for already canceling migration', () => {
        component.migration = {
            ...mockRunningMigration,
            canceling: true,
        } as MigrationStatus
        fixture.detectChanges()

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        expect(cancelButton.nativeElement.disabled).toBeTrue()
    })

    it('should display author information', () => {
        component.migration = {
            ...mockRunningMigration,
            authorId: 1,
            authorLogin: 'admin',
        } as MigrationStatus
        fixture.detectChanges()

        const content = fixture.debugElement.nativeElement.textContent
        expect(content).toContain('Started by')
        expect(content).toContain('admin')
    })

    it('should show appropriate status tag for each migration state', () => {
        // Running
        component.migration = mockRunningMigration as MigrationStatus
        fixture.detectChanges()
        let tagElement = fixture.debugElement.query(By.directive(Tag))
        expect(tagElement).toBeTruthy()
        let tag = tagElement.componentInstance as Tag
        expect(tag.severity).toBe('info')
        expect(tag.value).toBe('Running')

        // Canceling
        component.migration = { ...mockRunningMigration, canceling: true } as MigrationStatus
        fixture.detectChanges()
        tagElement = fixture.debugElement.query(By.directive(Tag))
        expect(tagElement).toBeTruthy()
        tag = tagElement.componentInstance as Tag
        expect(tag.severity).toBe('warning')
        expect(tag.value).toBe('Canceling')

        // Completed
        component.migration = mockCompletedMigration as MigrationStatus
        fixture.detectChanges()
        tagElement = fixture.debugElement.query(By.directive(Tag))
        expect(tagElement).toBeTruthy()
        tag = tagElement.componentInstance as Tag
        expect(tag.severity).toBe('success')
        expect(tag.value).toBe('Completed')

        // Failed
        component.migration = mockFailedMigration as MigrationStatus
        fixture.detectChanges()
        tagElement = fixture.debugElement.query(By.directive(Tag))
        expect(tagElement).toBeTruthy()
        tag = tagElement.componentInstance as Tag
        expect(tag.severity).toBe('danger')
        expect(tag.value).toBe('Failed')
    })
})
