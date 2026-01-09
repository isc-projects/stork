import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { HostsTableComponent } from './hosts-table.component'
import { Router, provideRouter } from '@angular/router'
import { HostsPageComponent } from '../hosts-page/hosts-page.component'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { InputNumber } from 'primeng/inputnumber'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { HttpErrorResponse, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { DHCPService, Host, LocalHost } from '../backend'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'
import { FilterMetadata } from 'primeng/api/filtermetadata'

describe('HostsTableComponent', () => {
    let component: HostsTableComponent
    let fixture: ComponentFixture<HostsTableComponent>
    let dhcpService: DHCPService
    let getHostsSpy: jasmine.Spy
    let startMigrationSpy: jasmine.Spy
    let router: Router

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                ConfirmationService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([
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

    it('should group the hosts by daemon', () => {
        // Arrange
        const hosts: Partial<Host>[] = [
            { id: 1, localHosts: [{ daemonId: 11 }] },
            { id: 2, localHosts: [{ daemonId: 22 }, { daemonId: 22 }, { daemonId: 33 }] },
            { id: 3, localHosts: [{ daemonId: 11 }, { daemonId: 22 }] },
        ]

        // Act
        component.hosts = hosts as Host[]

        // Assert
        expect(component.localHostsGroupedByDaemon[1].length).toBe(1)
        expect(component.localHostsGroupedByDaemon[1][0].length).toBe(1)
        expect(component.localHostsGroupedByDaemon[1][0][0].daemonId).toBe(11)

        expect(component.localHostsGroupedByDaemon[2].length).toBe(2)
        expect(component.localHostsGroupedByDaemon[2][0].length).toBe(2)
        expect(component.localHostsGroupedByDaemon[2][0][0].daemonId).toBe(22)
        expect(component.localHostsGroupedByDaemon[2][0][1].daemonId).toBe(22)
        expect(component.localHostsGroupedByDaemon[2][1].length).toBe(1)
        expect(component.localHostsGroupedByDaemon[2][1][0].daemonId).toBe(33)

        expect(component.localHostsGroupedByDaemon[3][0].length).toBe(1)
        expect(component.localHostsGroupedByDaemon[3][0][0].daemonId).toBe(11)
        expect(component.localHostsGroupedByDaemon[3][1].length).toBe(1)
        expect(component.localHostsGroupedByDaemon[3][1][0].daemonId).toBe(22)
    })

    it('should detect local hosts state', () => {
        // Arrange
        const zero = []

        const single: LocalHost[] = [
            {
                daemonId: 1,
                daemonName: "dhcp4",
                optionsHash: 'hash1',
                clientClasses: ['class1'],
                nextServer: 'srv1',
                serverHostname: 'host1',
                bootFileName: 'boot1',
            },
        ]

        const conflict: LocalHost[] = [
            {
                daemonId: 1,
                daemonName: "dhcp4",
                optionsHash: 'hash1',
                clientClasses: ['class1'],
                nextServer: 'srv1',
                serverHostname: 'host1',
                bootFileName: 'boot1',
            },
            {
                daemonId: 1,
                daemonName: "dhcp4",
                optionsHash: 'hash2',
                clientClasses: ['class2'],
                nextServer: 'srv2',
                serverHostname: 'host2',
                bootFileName: 'boot2',
            },
        ]

        const duplicate: LocalHost[] = [
            {
                daemonId: 1,
                daemonName: "dhcp4",
                optionsHash: 'hash1',
                clientClasses: ['class1'],
                nextServer: 'srv1',
                serverHostname: 'host1',
                bootFileName: 'boot1',
            },
            {
                daemonId: 1,
                daemonName: "dhcp4",
                optionsHash: 'hash1',
                clientClasses: ['class1'],
                nextServer: 'srv1',
                serverHostname: 'host1',
                bootFileName: 'boot1',
            },
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
            machineId: { value: 5 },
            daemonId: { value: 1 },
            subnetId: { value: 2 },
            keaSubnetId: { value: 7 },
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

        expect(dhcpService.startHostsMigration).toHaveBeenCalledWith(5, 1, 2, 7, 'foo', true)
    }))

    it('should extract filter entries properly', () => {
        // Empty filter. Conflict is set to true by default.
        component.table.filters = {
            conflict: { value: null },
        }
        expect(component.migrationFilterEntries).toEqual([['Conflict', 'false']])

        component.table.filters = {
            machineId: { value: 42 },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['Conflict', 'false'],
            ['Machine ID', '42'],
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
            machineId: { value: 1 },
            subnetId: { value: 1 },
            keaSubnetId: { value: 10 },
            isGlobal: { value: false },
            text: { value: 'foo' },
        }
        expect(component.migrationFilterEntries).toEqual([
            ['Conflict', 'false'],
            ['Is Global', 'false'],
            ['Kea Subnet ID', '10'],
            ['Machine ID', '1'],
            ['Subnet ID', '1'],
            ['Text', 'foo'],
        ])
    })

    it('should not filter the table by numeric input with value zero', fakeAsync(() => {
        // Arrange
        const inputNumbers = fixture.debugElement.queryAll(By.directive(InputNumber))
        expect(inputNumbers).toBeTruthy()
        expect(inputNumbers.length).toEqual(4)

        // Act
        component.table.clear()
        tick()
        fixture.detectChanges()
        inputNumbers[0].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // machineId
        tick(300)
        fixture.detectChanges()
        inputNumbers[1].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // daemonId
        tick(300)
        fixture.detectChanges()
        inputNumbers[2].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // subnetId
        tick(300)
        fixture.detectChanges()
        inputNumbers[3].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // keaSubnetId
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(dhcpService.getHosts).toHaveBeenCalled()
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: 1,
                daemonId: 1,
                subnetId: 1,
                keaSubnetId: 1,
                isGlobal: null,
                conflict: null,
                text: null,
            },
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
                localHosts: [{ daemonId: 1, daemonName: "dhcp4", dataSource: 'config' } as LocalHost],
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
                localHosts: [{ daemonId: 2, daemonName: "dhcp4", dataSource: 'config' } as LocalHost],
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
                localHosts: [{ daemonId: 3, daemonName: "dhcp4", dataSource: 'config' } as LocalHost],
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

    it('hosts list should be filtered by machineId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ daemonId: 1, daemonName: "dhcp4", dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(2, <FilterMetadata>component.table.filters['machineId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: 2,
                daemonId: null,
                subnetId: null,
                keaSubnetId: null,
                isGlobal: null,
                conflict: null,
                text: null,
            },
        })
    }))

    it('hosts list should be filtered by subnetId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ daemonId: 1, dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(89, <FilterMetadata>component.table.filters['subnetId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: null,
                subnetId: 89,
                keaSubnetId: null,
                isGlobal: null,
                conflict: null,
                text: null,
            },
        })
    }))

    it('hosts list should be filtered by conflicts', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ daemonId: 1, dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(true, <FilterMetadata>component.table.filters['conflict'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: null,
                subnetId: null,
                keaSubnetId: null,
                isGlobal: null,
                conflict: true,
                text: null,
            },
        })
    }))

    it('hosts list should be filtered by non-conflicts', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ daemonId: 1, dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(false, <FilterMetadata>component.table.filters['conflict'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: null,
                subnetId: null,
                keaSubnetId: null,
                isGlobal: null,
                conflict: false,
                text: null,
            },
        })
    }))

    it('hosts list should be filtered by keaSubnetId', fakeAsync(() => {
        component.hosts = [{ id: 1, localHosts: [{ daemonId: 1, daemonName: "dhcp4", dataSource: 'config' }] }]
        fixture.detectChanges()

        getHostsSpy.and.callThrough()

        component.filterTable(101, <FilterMetadata>component.table.filters['keaSubnetId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: null,
                subnetId: null,
                keaSubnetId: 101,
                isGlobal: null,
                conflict: null,
                text: null,
            },
        })
    }))

    it('should group the local hosts by daemonId', () => {
        const host = {
            id: 42,
            localHosts: [
                {
                    daemonId: 31,
                },
                {
                    daemonId: 32,
                },
                {
                    daemonId: 33,
                },
                {
                    daemonId: 21,
                },
                {
                    daemonId: 22,
                },
                {
                    daemonId: 11,
                },
            ],
        } as Host

        component.hosts = [host]
        const groups = component.localHostsGroupedByDaemon[host.id]

        expect(groups.length).toBe(6)
        const daemonIds = groups.map((g) => g[0].daemonId).sort()
        expect(daemonIds).toEqual([11, 21, 22, 31, 32, 33])

        const groupByDaemon = Object.fromEntries(groups.map((g) => [g[0].daemonId, g]))
        expect(groupByDaemon[31].length).toBe(1)
        expect(groupByDaemon[21].length).toBe(1)
        expect(groupByDaemon[11].length).toBe(1)
    })

    it('should recognize the state of local hosts', () => {
        // Conflict
        let localHosts = [
            {
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                daemonId: 2,
                nextServer: 'bar',
            },
        ] as LocalHost[]

        let state = component.getLocalHostsState(localHosts)
        expect(state).toBe('conflict')

        // Duplicate
        localHosts = [
            {
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                daemonId: 2,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.getLocalHostsState(localHosts)
        expect(state).toBe('duplicate')

        // Null
        localHosts = [
            {
                daemonId: 1,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.getLocalHostsState(localHosts)
        expect(state).toBeNull()
    })

    it('host table should have valid daemon name and daemon link', () => {
        component.hosts = [
            {
                id: 1,
                localHosts: [{ daemonId: 1, daemonName: 'frog', dataSource: 'config' } as unknown as LocalHost],
            } as Host,
        ]
        fixture.detectChanges()
        // Table rows have ids created by appending host id to the host-row- string.
        const row = fixture.debugElement.query(By.css('#host-row-1'))
        // There should be 6 table cells in the row.
        expect(row.children.length).toBe(6)
        // The last one includes the daemon name and link.
        const daemonTd = row.children[5]
        const daemonLink = daemonTd.query(By.css('a'))
        expect(daemonLink.nativeElement.textContent).toContain('frog')
        expect(daemonLink.properties.hasOwnProperty('pathname')).toBeTrue()
        expect(daemonLink.properties.pathname).toBe('/daemons/1')
        // Data source labels are still rendered.
        expect(daemonTd.query(By.css('app-host-data-source-label'))).toBeTruthy()
    })
})
