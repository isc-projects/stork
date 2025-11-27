import { ComponentFixture, fakeAsync, flush, TestBed, tick } from '@angular/core/testing'

import { SubnetsTableComponent } from './subnets-table.component'
import { InputNumber } from 'primeng/inputnumber'
import { MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideRouter } from '@angular/router'
import { SubnetsPageComponent } from '../subnets-page/subnets-page.component'
import { DHCPService, Subnets } from '../backend'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { FilterMetadata } from 'primeng/api/filtermetadata'

describe('SubnetsTableComponent', () => {
    let component: SubnetsTableComponent
    let fixture: ComponentFixture<SubnetsTableComponent>
    let dhcpApi: DHCPService
    let fakeResponse: any

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([
                    {
                        path: 'dhcp/subnets',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/subnets/all',
                    },
                    {
                        path: 'dhcp/subnets/:id',
                        component: SubnetsPageComponent,
                    },
                ]),
            ],
        }).compileComponents()

        dhcpApi = TestBed.inject(DHCPService)
        fixture = TestBed.createComponent(SubnetsTableComponent)
        component = fixture.componentInstance
        fakeResponse = {
            items: [
                {
                    clientClass: 'class-00-00',
                    id: 5,
                    localSubnets: [
                        {
                            appId: 27,
                            appName: 'kea@localhost',
                            id: 1,
                            machineAddress: 'localhost',
                            machineHostname: 'lv-pc',
                            pools: [
                                {
                                    pool: '1.0.0.4-1.0.255.254',
                                },
                            ],
                        },
                        {
                            appId: 28,
                            appName: 'kea2@localhost',
                            // Misconfiguration,  all local subnets in a
                            // subnet should share the same subnet ID. In
                            // this case, we display a value from the first
                            // local subnet.
                            id: 2,
                            machineAddress: 'host',
                            machineHostname: 'lv-pc2',
                            pools: [
                                {
                                    pool: '1.0.0.4-1.0.255.254',
                                },
                            ],
                        },
                    ],
                    stats: {
                        'assigned-addresses':
                            '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                        'total-addresses':
                            '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                        'declined-addresses': '-2',
                    },
                    statsCollectedAt: '2022-01-19T12:10:22.513Z',
                    subnet: '1.0.0.0/16',
                },
                {
                    clientClass: 'class-00-01',
                    id: 42,
                    localSubnets: [
                        {
                            appId: 27,
                            appName: 'kea@localhost',
                            machineAddress: 'localhost',
                            machineHostname: 'lv-pc',
                            pools: [
                                {
                                    pool: '1.1.0.4-1.1.255.254',
                                },
                            ],
                        },
                    ],
                    statsCollectedAt: null,
                    subnet: '1.1.0.0/16',
                },
                {
                    id: 67,
                    localSubnets: [
                        {
                            id: 4,
                            appId: 28,
                            appName: 'ha@localhost',
                            machineAddress: 'localhost',
                            machineHostname: 'ha-cluster-1',
                        },
                        {
                            id: 4,
                            appId: 28,
                            appName: 'ha@localhost',
                            machineAddress: 'localhost',
                            machineHostname: 'ha-cluster-2',
                        },
                        {
                            id: 4,
                            appId: 28,
                            appName: 'ha@localhost',
                            machineAddress: 'localhost',
                            machineHostname: 'ha-cluster-3',
                        },
                    ],
                    statsCollectedAt: '2022-01-16T14:16:00.000Z',
                    subnet: '1.1.1.0/24',
                },
            ],
            total: 10496,
        }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display links to grafana', fakeAsync(() => {
        const subnets: Subnets = {
            items: [
                {
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            id: 1,
                            machineHostname: 'foo',
                        },
                    ],
                },
                {
                    subnet: '192.0.3.0/24',
                    localSubnets: [
                        {
                            id: 2,
                            machineHostname: 'foo',
                        },
                    ],
                },
                {
                    subnet: '2001:db8:1::/64',
                    localSubnets: [
                        {
                            id: 3,
                            machineHostname: 'foo',
                        },
                    ],
                },
            ],
            total: 3,
        }
        spyOn(dhcpApi, 'getSubnets').and.returnValue(of(subnets as any))
        component.grafanaUrl = 'http://localhost:3000/'
        component.grafanaDhcp4DashboardId = 'hRf18FvWz'
        component.grafanaDhcp6DashboardId = 'AQPHKJUGz'

        const loadMetadata = component.table.createLazyLoadMetadata()
        component.loadData(loadMetadata)
        tick()
        fixture.detectChanges()

        const grafanaIcons = fixture.debugElement.queryAll(By.css('a .pi-chart-line'))
        expect(grafanaIcons?.length).toBe(3)

        for (let grafanaIcon of grafanaIcons) {
            const parent = grafanaIcon.nativeElement.parentElement
            expect(parent.tagName).toBe('A')
            const href = parent.getAttribute('href')
            const title = parent.getAttribute('title')
            if (title.includes('subnet 3')) {
                expect(href).toContain('AQPHKJUGz')
            } else {
                expect(href).toContain('hRf18FvWz')
            }
        }
    }))

    it('should recognize that there are subnets with names', () => {
        // No subnets.
        component.dataCollection = []
        expect(component.isAnySubnetWithNameVisible).toBeFalse()

        // Subnet without names.
        component.dataCollection = [{ localSubnets: [{ userContext: {} }] }]
        expect(component.isAnySubnetWithNameVisible).toBeFalse()

        // Subnet with names.
        component.dataCollection = [
            {
                localSubnets: [
                    {
                        userContext: { 'subnet-name': 'foo' },
                    },
                ],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        // Subnet with names and without names.
        component.dataCollection = [
            {
                localSubnets: [{ userContext: { 'subnet-name': 'foo' } }],
            },
            {
                localSubnets: [{ userContext: {} }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        // Subnet with multiple names.
        component.dataCollection = [
            {
                localSubnets: [{ userContext: { 'subnet-name': 'foo' } }],
            },
            {
                localSubnets: [{ userContext: { 'subnet-name': 'bar' } }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        component.dataCollection = [
            {
                localSubnets: [{ userContext: {} }],
            },
            {
                localSubnets: [{ userContext: { 'subnet-name': 'bar' } }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()
    })

    it('should recognize that there are subnets with multiple names', () => {
        // No local subnets.
        let subnet = {}
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Subnet without names.
        subnet = { localSubnets: [{ userContext: {} }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Single local subnet.
        subnet = { localSubnets: [{ userContext: { 'subnet-name': 'foo' } }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Multiple local subnets with the same name.
        subnet = {
            localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: { 'subnet-name': 'foo' } }],
        }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Multiple subnets with different names.
        subnet = {
            localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: { 'subnet-name': 'bar' } }],
        }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeTrue()

        // Multiple subnets. One with a name, one without.
        subnet = { localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: {} }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeTrue()
    })

    it('should not filter the table by numeric input with value zero', fakeAsync(() => {
        // Arrange
        const getSubnetsSpy = spyOn(dhcpApi, 'getSubnets').and.returnValue(of({ items: [], total: 0 }) as any)
        const inputNumbers = fixture.debugElement.queryAll(By.directive(InputNumber))
        expect(inputNumbers).toBeTruthy()
        expect(inputNumbers.length).toEqual(2)
        spyOn(component, 'filterTable')

        // Act
        component.table.clear()
        tick(300)
        fixture.detectChanges()
        inputNumbers[0].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // appId
        tick(300)
        fixture.detectChanges()
        inputNumbers[1].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // subnetId
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(getSubnetsSpy).toHaveBeenCalled()
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(component.filterTable).toHaveBeenCalledWith(1, component.table.filters['appId'] as FilterMetadata)
        expect(component.filterTable).toHaveBeenCalledWith(1, component.table.filters['subnetId'] as FilterMetadata)
        flush()
    }))

    it('should display the Kea subnet ID', async () => {
        // Act
        spyOn(dhcpApi, 'getSubnets').and.returnValue(of(fakeResponse as any))
        const metadata = component.table.createLazyLoadMetadata()
        component.loadData(metadata)
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        const cells = fixture.debugElement.queryAll(By.css('table tbody tr td:last-child'))
        expect(cells.length).toBe(3)
        const cellValues = cells.map((c) => (c.nativeElement as HTMLElement).textContent.trim())
        // First subnet has various Kea subnet IDs.
        expect(cellValues).toContain('1  2 Inconsistent IDs')
        // Second subnet misses the Kea subnet ID.
        expect(cellValues).toContain('')
        // Third subnet has identical Kea subnet IDs.
        expect(cellValues).toContain('4')
    })

    it('should convert statistics to big integers', async () => {
        // Act
        spyOn(dhcpApi, 'getSubnets').and.returnValue(of(fakeResponse as any))
        const metadata = component.table.createLazyLoadMetadata()
        component.loadData(metadata)
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(component.dataCollection).toBeTruthy()
        expect(component.dataCollection.length).toBeGreaterThan(0)
        const stats: { [key: string]: BigInt } = component.dataCollection[0].stats as any
        expect(stats['assigned-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['total-addresses']).toBe(
            BigInt(
                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890'
            )
        )
        expect(stats['declined-addresses']).toBe(BigInt('-2'))
    })
})
