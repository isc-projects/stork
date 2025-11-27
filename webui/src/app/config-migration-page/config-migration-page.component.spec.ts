import { ComponentFixture, TestBed, fakeAsync, flush, tick } from '@angular/core/testing'
import { HttpEvent, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfigMigrationPageComponent } from './config-migration-page.component'
import { DHCPService, MigrationStatus } from '../backend'
import { ConfirmationService, MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { ActivatedRoute, ParamMap, provideRouter } from '@angular/router'
import { BehaviorSubject, of, throwError, Observable } from 'rxjs'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MockParamMap } from '../utils'

describe('ConfigMigrationPageComponent', () => {
    let component: ConfigMigrationPageComponent
    let fixture: ComponentFixture<ConfigMigrationPageComponent>
    let dhcpApi: jasmine.SpyObj<DHCPService>
    let messageService: MessageService
    let paramMapSubject: BehaviorSubject<ParamMap>

    /**
     * Wraps response in HttpEvent type.
     */
    function wrapInHttpResponse<T>(body: T): Observable<HttpEvent<T>> {
        return of(body as any)
    }

    /**
     * Wraps empty response in HttpEvent type.
     */
    function wrapEmptyResponse(): Observable<HttpEvent<any>> {
        return of({} as any)
    }

    const mockRunningMigration: MigrationStatus = {
        id: 1,
        startDate: new Date(2023, 5, 15, 10, 0, 0).toISOString(),
        endDate: null,
        canceling: false,
        errors: {
            total: 0,
            items: [],
        },
        elapsedTime: '10m',
        estimatedLeftTime: '5m',
        authorId: 1,
        authorLogin: 'admin',
        processedItemsCount: 50,
        totalItemsCount: 100,
    }

    const mockCompletedMigration: MigrationStatus = {
        ...mockRunningMigration,
        id: 2,
        endDate: new Date(2023, 5, 15, 10, 15, 0).toISOString(),
        processedItemsCount: 100,
        totalItemsCount: 100,
    }

    beforeEach(async () => {
        dhcpApi = jasmine.createSpyObj('DHCPService', [
            'getMigration',
            'getMigrations',
            'putMigration',
            'deleteFinishedMigrations',
        ])
        paramMapSubject = new BehaviorSubject(new MockParamMap({ id: 'all' }))

        await TestBed.configureTestingModule({
            providers: [
                { provide: DHCPService, useValue: dhcpApi },
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        paramMap: paramMapSubject.asObservable(),
                    },
                },
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([
                    { path: 'config-migrations/all', component: ConfigMigrationPageComponent },
                    { path: 'config-migrations/1', component: ConfigMigrationPageComponent },
                ]),
            ],
        }).compileComponents()

        messageService = TestBed.inject(MessageService)
        fixture = TestBed.createComponent(ConfigMigrationPageComponent)
        component = fixture.componentInstance
    })

    it('should create', () => {
        fixture.detectChanges()
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        fixture.detectChanges()

        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).toBeTruthy()

        const breadcrumbsComponent = breadcrumbsElement.componentInstance
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toBe('DHCP')
        expect(breadcrumbsComponent.items[1].label).toBe('Config Migrations')
    })

    xit('should open migration tab when route changes', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        dhcpApi.getMigration.and.returnValue(wrapInHttpResponse(mockRunningMigration))
        fixture.detectChanges()

        // Simulate navigation to specific migration
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getMigration).toHaveBeenCalledWith(1)
        // expect(component.tabs.length).toBe(2)
        expect(component.activeTabMigrationID()).toBe(1)
        // expect(component.tabItems[component.activeTabMigrationID()]).toEqual(mockRunningMigration)

        flush()
    }))

    it('should handle error when opening migration tab', fakeAsync(() => {
        dhcpApi.getMigration.and.returnValue(throwError(() => new Error('Failed to get migration')))
        let migration: MigrationStatus, error: any
        component
            .migrationProvider(1)
            .then((m) => (migration = m))
            .catch((e) => (error = e))
        tick()
        expect(dhcpApi.getMigration).toHaveBeenCalledWith(1)
        expect(migration).toBeUndefined()
        expect(error).toBeInstanceOf(Error)
    }))

    xit('should switch to existing tab without API call', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        dhcpApi.getMigration.and.returnValue(wrapInHttpResponse(mockRunningMigration))
        fixture.detectChanges()

        // Open tab first time
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()
        // expect(component.tabs.length).toBe(2)
        // expect(component.tabItems.length).toBe(2)
        // expect(component.tabItems[1]).toBe(mockRunningMigration)
        // expect(component.tabItems[1].id).toBe(1)

        // Reset spy count
        dhcpApi.getMigration.calls.reset()

        // Try to open same tab again
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getMigration).not.toHaveBeenCalled()
        // expect(component.tabs.length).toBe(2)
        expect(component.activeTabMigrationID()).toBe(1)
    }))

    xit('should close migration tab', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        dhcpApi.getMigration.and.returnValue(wrapInHttpResponse(mockRunningMigration))
        fixture.detectChanges()

        // Open tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // expect(component.tabs.length).toBe(2)

        // Close tab
        // component.closeTab(1)
        fixture.detectChanges()

        // expect(component.tabs.length).toBe(1)
        expect(component.activeTabMigrationID()).toBe(0)
        // expect(component.tabItems[component.activeTabMigrationID]).toEqual({})

        flush()
    }))

    xit('should not allow closing the main tab', () => {
        // TODO: this test should be moved away from Karma tests.
        fixture.detectChanges()

        component.tabView().closeTab(0)
        fixture.detectChanges()

        // expect(component.tabs.length).toBe(1)
        expect(component.activeTabMigrationID()).toBe(0)
    })

    it('should cancel migration', fakeAsync(() => {
        dhcpApi.getMigration.and.returnValue(wrapInHttpResponse(mockRunningMigration))
        const canceledMigration: MigrationStatus = {
            ...mockRunningMigration,
            canceling: true,
        }
        dhcpApi.putMigration.and.returnValue(wrapInHttpResponse(canceledMigration))
        fixture.detectChanges()

        // spyOn(component.alteredStatuses, 'next')

        // Open tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // Cancel migration
        component.onCancelMigration(1)
        tick()
        fixture.detectChanges()

        expect(dhcpApi.putMigration).toHaveBeenCalledWith(1)
        // expect(component.tabItems[1].canceling).toBeTrue()
        // expect(component.alteredStatuses.next).toHaveBeenCalledWith(canceledMigration)
    }))

    it('should handle error when canceling migration', fakeAsync(() => {
        dhcpApi.getMigration.and.returnValue(wrapInHttpResponse(mockRunningMigration))
        dhcpApi.putMigration.and.returnValue(throwError(() => new Error('Failed to cancel')))
        spyOn(messageService, 'add')
        fixture.detectChanges()

        // Open tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // Try to cancel
        component.onCancelMigration(1)
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Failed to cancel migration',
            })
        )
    }))

    it('should clean up finished migrations', fakeAsync(() => {
        // Setup tabs with both running and completed migrations
        dhcpApi.getMigration.and.returnValues(
            wrapInHttpResponse(mockCompletedMigration),
            wrapInHttpResponse(mockRunningMigration)
        )
        dhcpApi.deleteFinishedMigrations.and.returnValue(wrapEmptyResponse())
        fixture.detectChanges()

        // Open completed migration tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        // Open running migration tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // Clean up
        component.onClearFinishedMigrations()
        tick()
        fixture.detectChanges()

        expect(dhcpApi.deleteFinishedMigrations).toHaveBeenCalled()
    }))

    it('should handle error when cleaning up finished migrations', fakeAsync(() => {
        dhcpApi.deleteFinishedMigrations.and.returnValue(throwError(() => new Error('error happened')))
        spyOn(messageService, 'add')

        component.onClearFinishedMigrations()
        tick()

        expect(dhcpApi.deleteFinishedMigrations).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Failed to clean up finished migrations',
            })
        )
    }))

    it('should handle error when cleaning up migrations', fakeAsync(() => {
        dhcpApi.deleteFinishedMigrations.and.returnValue(throwError(() => new Error('Failed to clean up')))
        spyOn(messageService, 'add')
        fixture.detectChanges()

        component.onClearFinishedMigrations()
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Failed to clean up finished migrations',
            })
        )
    }))

    xit('should refresh migration status', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        const updatedMigration: MigrationStatus = {
            ...mockRunningMigration,
            processedItemsCount: 75,
            totalItemsCount: 100,
        }
        dhcpApi.getMigration.and.returnValues(
            wrapInHttpResponse(mockRunningMigration),
            wrapInHttpResponse(updatedMigration)
        )
        fixture.detectChanges()

        // spyOn(component.alteredStatuses, 'next')

        // Open tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // Reset spy count
        dhcpApi.getMigration.calls.reset()

        // Refresh status
        // component.onRefreshMigration(1)
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getMigration).toHaveBeenCalledWith(1)
        // expect(component.tabItems[1].processedItemsCount).toBe(75)
        // expect(component.alteredStatuses.next).toHaveBeenCalledWith(updatedMigration)
    }))

    xit('should handle error when refreshing migration status', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        dhcpApi.getMigration.and.returnValues(
            wrapInHttpResponse(mockRunningMigration),
            throwError(() => new Error('Failed to refresh'))
        )
        spyOn(messageService, 'add')
        fixture.detectChanges()

        // spyOn(component.alteredStatuses, 'next')

        // Open tab
        paramMapSubject.next(new MockParamMap({ id: '1' }))
        tick()
        fixture.detectChanges()

        // Reset spy count
        dhcpApi.getMigration.calls.reset()

        // Try to refresh
        // component.onRefreshMigration(1)
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Failed to refresh migration status',
            })
        )
        // expect(component.alteredStatuses.next).not.toHaveBeenCalled()
    }))
})
