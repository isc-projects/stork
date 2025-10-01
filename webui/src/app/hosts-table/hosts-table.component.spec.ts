import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { HostsTableComponent } from './hosts-table.component'
import { TableModule } from 'primeng/table'
import { Router, RouterModule } from '@angular/router'
import { HostsPageComponent } from '../hosts-page/hosts-page.component'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { InputNumber, InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { TagModule } from 'primeng/tag'
import { HttpErrorResponse, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ManagedAccessDirective } from '../managed-access.directive'
import { ConfirmDialog, ConfirmDialogModule } from 'primeng/confirmdialog'
import { DHCPService, Host, LocalHost } from '../backend'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'
import { FloatLabelModule } from 'primeng/floatlabel'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'
import { HostDataSourceLabelComponent } from '../host-data-source-label/host-data-source-label.component'
import { IdentifierComponent } from '../identifier/identifier.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ByteCharacterComponent } from '../byte-character/byte-character.component'

describe('HostsTableComponent', () => {
    let component: HostsTableComponent
    let fixture: ComponentFixture<HostsTableComponent>
    let dhcpService: DHCPService
    let getHostsSpy: jasmine.Spy
    let startMigrationSpy: jasmine.Spy
    let router: Router

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [
                HostsTableComponent,
                HelpTipComponent,
                PluralizePipe,
                IdentifierComponent,
                HostDataSourceLabelComponent,
                EntityLinkComponent,
                ByteCharacterComponent,
            ],
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
                PopoverModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
                ManagedAccessDirective,
                ConfirmDialogModule,
                FloatLabelModule,
                TriStateCheckboxComponent,
                IconFieldModule,
                InputIconModule,
            ],
            providers: [
                MessageService,
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        }).compileComponents()

        dhcpService = TestBed.inject(DHCPService)
        router = TestBed.inject(Router)
        getHostsSpy = spyOn(dhcpService, 'getHosts')
        startMigrationSpy = spyOn(dhcpService, 'startHostsMigration')
        spyOn(router, 'navigate')
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.table, 'saveState').and.callFake(() => {})
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
        startMigrationSpy.and.returnValue(of({}) as any)

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
        confirmDialog.onAccept()
        tick()

        expect(dhcpService.startHostsMigration).toHaveBeenCalledWith(1, null, null, 'foo', true)
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
            ['App ID', '42'],
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
            ['App ID', '1'],
            ['Conflict', 'false'],
            ['Is Global', 'false'],
            ['Subnet ID', '1'],
            ['Text', 'foo'],
        ])
    })

    it('should not filter the table by numeric input with value zero', fakeAsync(() => {
        // Arrange
        const inputNumbers = fixture.debugElement.queryAll(By.directive(InputNumber))
        expect(inputNumbers).toBeTruthy()
        expect(inputNumbers.length).toEqual(3)

        // Act
        component.table.clear()
        tick()
        fixture.detectChanges()
        inputNumbers[0].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // appId
        tick(300)
        fixture.detectChanges()
        inputNumbers[1].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // subnetId
        tick(300)
        fixture.detectChanges()
        inputNumbers[2].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // keaSubnetId
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(dhcpService.getHosts).toHaveBeenCalled()
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: { appId: 1, subnetId: 1, keaSubnetId: 1, isGlobal: null, conflict: null, text: null },
        })
    }))

    it('should display well formatted host identifiers', () => {
        // Create a list with three hosts. One host uses a duid convertible
        // to a textual format. Another host uses a hw-address which is
        // by default displayed in the hex format. Third host uses a
        // flex-id which is not convertible to a textual format.
        component.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '61:62:63:64',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'hw-address',
                        idHexValue: '51:52:53:54:55:56',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 3,
                hostIdentifiers: [
                    {
                        idType: 'flex-id',
                        idHexValue: '01:02:03:04:05',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 3,
                        appName: 'lion',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // There should be 3 hosts listed.
        const identifierEl = fixture.debugElement.queryAll(By.css('app-identifier'))
        expect(identifierEl.length).toBe(3)

        // Each host identifier should be a link.
        const firstIdEl = identifierEl[0].query(By.css('a'))
        expect(firstIdEl).toBeTruthy()
        // The DUID is displayed by default as a hex.
        expect(firstIdEl.nativeElement.textContent).toContain('duid=(61:62:63:64)')
        expect(firstIdEl.attributes.href).toBe('/dhcp/hosts/1')

        const secondIdEl = identifierEl[1].query(By.css('a'))
        expect(secondIdEl).toBeTruthy()
        // The HW address is convertible but by default should be in hex format.
        expect(secondIdEl.nativeElement.textContent).toContain('hw-address=(51:52:53:54:55:56)')
        expect(secondIdEl.attributes.href).toBe('/dhcp/hosts/2')

        const thirdIdEl = identifierEl[2].query(By.css('a'))
        expect(thirdIdEl).toBeTruthy()
        // The flex-id is not convertible to text so should be in hex format.
        expect(thirdIdEl.nativeElement.textContent).toContain('flex-id=(\\0x01\\0x02\\0x03\\0x04\\0x05)')
        expect(thirdIdEl.attributes.href).toBe('/dhcp/hosts/3')
    })

    it('should contain a refresh button', fakeAsync(() => {
        const refreshBtn = fixture.debugElement.query(By.css('[label="Refresh List"] button'))
        expect(refreshBtn).toBeTruthy()
        spyOn(component, 'loadData')

        getHostsSpy.and.returnValue(throwError(() => new HttpErrorResponse({ status: 404 })))
        refreshBtn.nativeElement.click()
        tick()
        fixture.detectChanges()
        expect(component.loadData).toHaveBeenCalled()
    }))

    it('hosts list should be filtered by appId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(2, <FilterMetadata>component.table.filters['appId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: { appId: 2, subnetId: null, keaSubnetId: null, isGlobal: null, conflict: null, text: null },
        })
    }))

    it('hosts list should be filtered by subnetId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(89, <FilterMetadata>component.table.filters['subnetId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: { appId: null, subnetId: 89, keaSubnetId: null, isGlobal: null, conflict: null, text: null },
        })
    }))

    it('hosts list should be filtered by conflicts', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(true, <FilterMetadata>component.table.filters['conflict'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: { appId: null, subnetId: null, keaSubnetId: null, isGlobal: null, conflict: true, text: null },
        })
    }))

    it('hosts list should be filtered by non-conflicts', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(false, <FilterMetadata>component.table.filters['conflict'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                appId: null,
                subnetId: null,
                keaSubnetId: null,
                isGlobal: null,
                conflict: false,
                text: null,
            },
        })
    }))

    it('hosts list should be filtered by keaSubnetId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(101, <FilterMetadata>component.table.filters['keaSubnetId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: { appId: null, subnetId: null, keaSubnetId: 101, isGlobal: null, conflict: null, text: null },
        })
    }))

    it('should group the local hosts by appId', () => {
        const host = {
            id: 42,
            localHosts: [
                {
                    appId: 3,
                    daemonId: 31,
                },
                {
                    appId: 3,
                    daemonId: 32,
                },
                {
                    appId: 3,
                    daemonId: 33,
                },
                {
                    appId: 2,
                    daemonId: 21,
                },
                {
                    appId: 2,
                    daemonId: 22,
                },
                {
                    appId: 1,
                    daemonId: 11,
                },
            ],
        } as Host

        component.hosts = [host]
        const groups = component.localHostsGroupedByApp[host.id]

        expect(groups.length).toBe(3)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId
            expect(group.length).toBe(appId)
            for (let item of group) {
                expect(item.daemonId).toBeGreaterThan(10 * appId)
                expect(item.daemonId).toBeLessThan(10 * (appId + 1))
            }
        }
    })

    it('should recognize the state of local hosts', () => {
        // Conflict
        let localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                appId: 1,
                daemonId: 2,
                nextServer: 'bar',
            },
        ] as LocalHost[]

        let state = component.getLocalHostsState(localHosts)
        expect(state).toBe('conflict')

        // Duplicate
        localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                appId: 1,
                daemonId: 2,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.getLocalHostsState(localHosts)
        expect(state).toBe('duplicate')

        // Null
        localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.getLocalHostsState(localHosts)
        expect(state).toBeNull()
    })

    it('host table should have valid app name and app link', () => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()
        // Table rows have ids created by appending host id to the host-row- string.
        const row = fixture.debugElement.query(By.css('#host-row-1'))
        // There should be 6 table cells in the row.
        expect(row.children.length).toBe(6)
        // The last one includes the app name.
        const appNameTd = row.children[5]
        // The cell includes a link to the app.
        expect(appNameTd.children.length).toBe(1)
        const appLink = appNameTd.children[0]
        expect(appLink.nativeElement.textContent).toBe('frog config')
        // Verify that the link to the app is correct.
        const appLinkAnchor = appLink.query(By.css('a'))
        expect(appLinkAnchor.properties.hasOwnProperty('pathname')).toBeTrue()
        expect(appLinkAnchor.properties.pathname).toBe('/apps/1')
    })
})
