import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'
import { HttpResponse, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

import { ConfigMigrationTableComponent } from './config-migration-table.component'
import { TableLazyLoadEvent } from 'primeng/table'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { DHCPService, MigrationStatuses } from '../backend'
import { Observable, of } from 'rxjs'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { By } from '@angular/platform-browser'

describe('ConfigMigrationTableComponent', () => {
    let component: ConfigMigrationTableComponent
    let fixture: ComponentFixture<ConfigMigrationTableComponent>
    let dhcpServiceSpy: jasmine.SpyObj<DHCPService>

    beforeEach(waitForAsync(() => {
        const spy = jasmine.createSpyObj('DHCPService', ['getMigrations'])

        // Setup the spy to return empty migrations list by default
        const emptyResponse: MigrationStatuses = {
            items: [],
            total: 0,
        }
        spy.getMigrations.and.returnValue(of(emptyResponse))

        TestBed.configureTestingModule({
            providers: [
                MessageService,
                { provide: DHCPService, useValue: spy },
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideNoopAnimations(),
            ],
        }).compileComponents()

        dhcpServiceSpy = TestBed.inject(DHCPService) as jasmine.SpyObj<DHCPService>
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigMigrationTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should call dhcpApi.getMigrations on loadData', () => {
        const event = { first: 0, rows: 10 } as TableLazyLoadEvent
        component.loadData(event)
        expect(dhcpServiceSpy.getMigrations).toHaveBeenCalledWith(0, 10)
    })

    it('should set migrations and totalRecords on successful loadData', async () => {
        // Setup test data
        const testMigrations = [
            { id: '1', progress: 0.5 },
            { id: '2', progress: 1.0 },
        ]

        dhcpServiceSpy.getMigrations.and.returnValue(
            of({
                items: testMigrations,
                total: 2,
            }) as any as Observable<HttpResponse<MigrationStatuses>>
        )

        // Call loadData
        const event = { first: 0, rows: 10 } as TableLazyLoadEvent
        component.loadData(event)

        // Wait for async operations
        fixture.detectChanges()
        await fixture.whenStable()

        // Verify the component state
        expect(component.dataCollection.length).toBe(2)
        expect(component.totalRecords).toBe(2)
    })

    it('should emit clearFinishedMigrationsRequest when onClearFinishedMigrations is called', fakeAsync(() => {
        spyOn(component.clearMigrations, 'emit')

        component.clearFinishedMigrations()
        fixture.whenRenderingDone()

        const dialog = fixture.debugElement.query(By.directive(ConfirmDialog))
        expect(dialog).not.toBeNull()
        const confirmDialog = dialog.componentInstance as ConfirmDialog
        expect(confirmDialog).not.toBeNull()
        confirmDialog.onAccept()
        tick()

        expect(component.clearMigrations.emit).toHaveBeenCalled()
    }))

    it('should emit cancelMigrationRequest with migrationId when onCancelMigration is called', () => {
        // Arrange
        spyOn(component.cancelMigration, 'emit')
        const migrationId = 1234

        // Act
        component.cancel(migrationId)

        // Assert
        expect(component.cancelMigration.emit).toHaveBeenCalledWith(migrationId)
    })
})
