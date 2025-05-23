import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { HostsTableComponent } from './hosts-table.component'
import { TableModule } from 'primeng/table'
import { RouterModule } from '@angular/router'
import { HostsPageComponent } from '../hosts-page/hosts-page.component'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { TagModule } from 'primeng/tag'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ManagedAccessDirective } from '../managed-access.directive'
import { ConfirmDialog, ConfirmDialogModule } from 'primeng/confirmdialog'
import { DHCPService, Host } from '../backend'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'

describe('HostsTableComponent', () => {
    let component: HostsTableComponent
    let fixture: ComponentFixture<HostsTableComponent>
    let dhcpServiceSpy: jasmine.SpyObj<DHCPService>

    beforeEach(waitForAsync(() => {
        const spy = jasmine.createSpyObj('DHCPService', ['startHostsMigration', 'getHosts'])

        TestBed.configureTestingModule({
            declarations: [HostsTableComponent, HelpTipComponent, PluralizePipe],
            imports: [
                TableModule,
                RouterModule.forRoot([
                    {
                        path: 'dhcp/hosts',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/hosts/all',
                    },
                    {
                        path: 'dhcp/hosts/:id',
                        component: HostsPageComponent,
                    },
                    {
                        path: 'config-migrations/:id',
                        redirectTo: 'dhcp/hosts/all',
                    },
                ]),
                ButtonModule,
                OverlayPanelModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
                ManagedAccessDirective,
                ConfirmDialogModule,
            ],
            providers: [
                MessageService,
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                { provide: DHCPService, useValue: spy },
            ],
        }).compileComponents()

        dhcpServiceSpy = TestBed.inject(DHCPService) as jasmine.SpyObj<DHCPService>
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should group the hosts by application', () => {
        // Arrange
        const hosts: Partial<Host>[] = [
            { id: 1, localHosts: [{ appId: 11 }] },
            { id: 2, localHosts: [{ appId: 22 }, { appId: 22 }, { appId: 33 }] },
            { id: 3, localHosts: [{ appId: 11 }, { appId: 22 }] },
        ]

        // Act
        component.hosts = hosts as Host[]

        // Assert
        expect(component.localHostsGroupedByApp[1].length).toBe(1)
        expect(component.localHostsGroupedByApp[1][0].length).toBe(1)
        expect(component.localHostsGroupedByApp[1][0][0].appId).toBe(11)

        expect(component.localHostsGroupedByApp[2].length).toBe(2)
        expect(component.localHostsGroupedByApp[2][0].length).toBe(2)
        expect(component.localHostsGroupedByApp[2][0][0].appId).toBe(22)
        expect(component.localHostsGroupedByApp[2][0][1].appId).toBe(22)
        expect(component.localHostsGroupedByApp[2][1].length).toBe(1)
        expect(component.localHostsGroupedByApp[2][1][0].appId).toBe(33)

        expect(component.localHostsGroupedByApp[3][0].length).toBe(1)
        expect(component.localHostsGroupedByApp[3][0][0].appId).toBe(11)
        expect(component.localHostsGroupedByApp[3][1].length).toBe(1)
        expect(component.localHostsGroupedByApp[3][1][0].appId).toBe(22)
    })

    it('should detect local hosts state', () => {
        // Arrange
        const zero = []

        const single = [{ appId: 1, bootFields: { field1: 'value1' }, clientClasses: ['class1'], dhcpOptions: {} }]

        const conflict = [
            { appId: 1, bootFields: { field1: 'value1' }, clientClasses: ['class1'], dhcpOptions: {} },
            { appId: 1, bootFields: { field1: 'value2' }, clientClasses: ['class2'], dhcpOptions: {} },
        ]

        const duplicate = [
            { appId: 1, bootFields: { field1: 'value1' }, clientClasses: ['class1'], dhcpOptions: {} },
            { appId: 1, bootFields: { field1: 'value1' }, clientClasses: ['class1'], dhcpOptions: {} },
        ]

        // Act
        const zeroState = component.getLocalHostsState(zero)
        const singleState = component.getLocalHostsState(single)
        const conflictState = component.getLocalHostsState(conflict)
        const duplicateState = component.getLocalHostsState(duplicate)

        // Assert
        expect(zeroState).toBeNull()
        expect(singleState).toBeNull()
        expect(conflictState).toBe('conflict')
        expect(duplicateState).toBe('duplicate')
    })

    it('should ask for confirmation before migrating hosts', fakeAsync(() => {
        dhcpServiceSpy.startHostsMigration.and.returnValue(of({}) as any)

        component.canStartMigration = true

        component.table.filters = {
            appId: { value: 1 },
            isGlobal: { value: true },
            text: { value: 'foo' },
        }

        component.migrateToDatabaseAsk()
        fixture.whenRenderingDone()

        const dialog = fixture.debugElement.query(By.directive(ConfirmDialog))
        expect(dialog).not.toBeNull()
        const confirmDialog = dialog.componentInstance as ConfirmDialog
        expect(confirmDialog).not.toBeNull()
        confirmDialog.accept()
        tick()

        expect(dhcpServiceSpy.startHostsMigration).toHaveBeenCalledWith(1, null, null, 'foo', true)
    }))

    it('should extract filter entries properly', () => {
        // Empty filter. Conflict is set to true by default.
        component.table.filters = {
            conflict: { value: null },
        }
        expect(component.migrationFilterEntries).toEqual([['Conflict', 'false']])

        component.table.filters = {
            appId: { value: 42 },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['App Id', '42'],
            ['Conflict', 'false'],
        ])

        component.table.filters = {
            isGlobal: { value: true },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['Conflict', 'false'],
            ['Is Global', 'true'],
        ])

        component.table.filters = {
            isGlobal: { value: false },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['Conflict', 'false'],
            ['Is Global', 'false'],
        ])

        component.table.filters = {
            appId: { value: 1 },
            subnetId: { value: 1 },
            isGlobal: { value: false },
            text: { value: 'foo' },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['App Id', '1'],
            ['Conflict', 'false'],
            ['Is Global', 'false'],
            ['Subnet Id', '1'],
            ['Text', 'foo'],
        ])
    })
})
