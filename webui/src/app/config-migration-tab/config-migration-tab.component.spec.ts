import { ComponentFixture, TestBed } from '@angular/core/testing'
import { ConfigMigrationTabComponent } from './config-migration-tab.component'
import { MigrationStatus } from '../backend'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { Table } from 'primeng/table'
import { ProgressBar } from 'primeng/progressbar'
import { Tag } from 'primeng/tag'
import { Fieldset } from 'primeng/fieldset'
import { provideRouter } from '@angular/router'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { AuthService } from '../auth.service'

describe('ConfigMigrationTabComponent', () => {
    let component: ConfigMigrationTabComponent
    let fixture: ComponentFixture<ConfigMigrationTabComponent>
    let authService: AuthService

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
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
                provideNoopAnimations(),
                provideRouter([]),
            ],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigMigrationTabComponent)
        component = fixture.componentInstance
        authService = fixture.debugElement.injector.get(AuthService)
        spyOn(authService, 'hasPrivilege').and.returnValue(true)
    })

    function render(migration: MigrationStatus = mockRunningMigration as MigrationStatus): void {
        component.migration = migration
        fixture.detectChanges()
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should emit refresh event when refresh button is clicked', () => {
        render()
        spyOn(component.refreshMigration, 'emit')

        const refreshButton = fixture.debugElement.query(By.css('button:not(.p-button-danger)'))
        refreshButton.nativeElement.click()

        expect(component.refreshMigration.emit).toHaveBeenCalledWith(1)
    })

    it('should emit event when cancel button is clicked', () => {
        render()
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
        render({
            ...mockRunningMigration,
            processedItemsCount: 6,
            totalItemsCount: 10,
        } as MigrationStatus)

        const progressBarElement = fixture.debugElement.query(By.directive(ProgressBar))
        expect(progressBarElement).toBeTruthy()
        const progressBar = progressBarElement.componentInstance as ProgressBar
        expect(progressBar.value).toBe(60)
    })

    it('should display 100% completion for finished migration', () => {
        render({
            ...mockCompletedMigration,
            processedItemsCount: 10,
            totalItemsCount: 10,
        } as MigrationStatus)

        const progressBarElement = fixture.debugElement.query(By.directive(ProgressBar))
        expect(progressBarElement).toBeTruthy()
        const progressBar = progressBarElement.componentInstance as ProgressBar
        expect(progressBar.value).toBe(100)
    })

    it('should display error items in table', () => {
        render(mockFailedMigration as MigrationStatus)

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

    it('should show time information for running migration', () => {
        render(mockRunningMigration as MigrationStatus)
        const content = fixture.nativeElement.textContent
        expect(content).toContain('Duration:10 minutes')
        expect(content).toContain('Estimated Left:5 minutes')
    })

    it('should show time information for completed migration', () => {
        render(mockCompletedMigration as MigrationStatus)
        const content = fixture.nativeElement.textContent
        expect(content).toContain('Duration:15 minutes')
        expect(content).not.toContain('Estimated Left')
    })

    it('should display general error in fieldset when present', () => {
        render(mockFailedMigration as MigrationStatus)

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
        render(mockCompletedMigration as MigrationStatus)

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        expect(cancelButton.nativeElement.disabled).toBeTrue()
    })

    it('should disable cancel button for already canceling migration', () => {
        render({
            ...mockRunningMigration,
            canceling: true,
        } as MigrationStatus)

        const cancelButton = fixture.debugElement.query(By.css('button.p-button-danger'))
        expect(cancelButton.nativeElement.disabled).toBeTrue()
    })

    it('should display author information', () => {
        render({
            ...mockRunningMigration,
            authorId: 1,
            authorLogin: 'admin',
        } as MigrationStatus)

        const content = fixture.debugElement.nativeElement.textContent
        expect(content).toContain('Started by')
        expect(content).toContain('admin')
    })

    it('should show running status tag', () => {
        render(mockRunningMigration as MigrationStatus)
        const tag = fixture.debugElement.query(By.directive(Tag)).componentInstance as Tag
        expect(tag.severity).toBe('info')
        expect(tag.value).toBe('Running')
    })

    it('should show canceling status tag', () => {
        render({ ...mockRunningMigration, canceling: true } as MigrationStatus)
        const tag = fixture.debugElement.query(By.directive(Tag)).componentInstance as Tag
        expect(tag.severity).toBe('warn')
        expect(tag.value).toBe('Canceling')
    })

    it('should show completed status tag', () => {
        render(mockCompletedMigration as MigrationStatus)
        const tag = fixture.debugElement.query(By.directive(Tag)).componentInstance as Tag
        expect(tag.severity).toBe('success')
        expect(tag.value).toBe('Completed')
    })

    it('should show failed status tag', () => {
        render(mockFailedMigration as MigrationStatus)
        const tag = fixture.debugElement.query(By.directive(Tag)).componentInstance as Tag
        expect(tag.severity).toBe('danger')
        expect(tag.value).toBe('Failed')
    })
})
