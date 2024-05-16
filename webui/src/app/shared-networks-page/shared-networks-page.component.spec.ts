import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { ActivatedRoute } from '@angular/router'
import { DHCPService, SharedNetwork, SharedNetworks } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { RouterTestingModule } from '@angular/router/testing'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { HttpEvent } from '@angular/common/http'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { MessageService } from 'primeng/api'
import { TabMenuModule } from 'primeng/tabmenu'
import { SharedNetworkTabComponent } from '../shared-network-tab/shared-network-tab.component'
import { FieldsetModule } from 'primeng/fieldset'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { DividerModule } from 'primeng/divider'
import { ChartModule } from 'primeng/chart'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { MockParamMap } from '../utils'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>
    let dhcpService: DHCPService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                BreadcrumbModule,
                ChartModule,
                DividerModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterTestingModule.withRoutes([
                    {
                        path: 'dhcp/shared-networks',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/shared-networks/all',
                    },
                    {
                        path: 'dhcp/shared-networks/:id',
                        component: SharedNetworksPageComponent,
                    },
                ]),
                TableModule,
                TabMenuModule,
                TooltipModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                BreadcrumbsComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                LocalNumberPipe,
                DelegatedPrefixBarComponent,
                PlaceholderPipe,
                SharedNetworksPageComponent,
                SharedNetworkTabComponent,
                SubnetBarComponent,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
            ],
            providers: [
                DHCPService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(new MockParamMap()),
                    },
                },
                RouterTestingModule,
            ],
        })

        dhcpService = TestBed.inject(DHCPService)
    }))

    beforeEach(() => {
        const fakeResponses: SharedNetworks[] = [
            {
                items: [
                    {
                        id: 1,
                        name: 'frog',
                        subnets: [
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
                                ],
                                subnet: '1.0.0.0/16',
                                statsCollectedAt: '2023-02-17T13:06:00.2134Z',
                                stats: {
                                    'assigned-addresses': '42',
                                    'total-addresses':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-addresses': '0',
                                },
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
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 2,
                        name: 'frog',
                        subnets: [
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
                                ],
                                subnet: '1.0.0.0/16',
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 3,
                        name: 'cat',
                        subnets: [
                            // Subnet represented by the double utilization bar.
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
                                    },
                                ],
                                subnet: 'fe80:1::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '24',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 10,
                                pdUtilization: 15,
                            },
                            // Subnet represented by the single NA utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 6,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:2::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 20,
                                pdUtilization: 0,
                            },
                            // Subnet represented by the single PD utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 7,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:3::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 0,
                                pdUtilization: 35,
                            },
                            // Subnet represented by the double utilization bar
                            {
                                clientClass: 'class-00-00',
                                id: 8,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 2,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:4::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 0,
                                pdUtilization: 0,
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
        ]
        spyOn(dhcpService, 'getSharedNetworks').and.returnValues(
            // The shared networks are fetched twice before the unit test starts.
            of(fakeResponses[0] as HttpEvent<SharedNetworks>),
            of(fakeResponses[0] as HttpEvent<SharedNetworks>),
            of(fakeResponses[1] as HttpEvent<SharedNetworks>),
            of(fakeResponses[2] as HttpEvent<SharedNetworks>)
        )

        spyOn(dhcpService, 'getSharedNetwork').and.returnValues(
            of(fakeResponses[0].items[0] as HttpEvent<SharedNetwork>)
        )

        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert shared network statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.networks[0].stats as any
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

    it('should convert subnet statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.networks[0].subnets[0].stats as any
        expect(stats['assigned-addresses']).toBe(BigInt('42'))
        expect(stats['total-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['declined-addresses']).toBe(BigInt('0'))
    })

    it('should not fail on empty statistics', async () => {
        // Act
        component.loadNetworks({})
        await fixture.whenStable()

        // Assert
        expect(component.networks[0].stats).toBeUndefined()
        // No throw
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Shared Networks')
    })

    it('should detect IPv6 subnets', () => {
        const networks: SharedNetwork[] = [
            {
                subnets: [{ subnet: '10.0.0.0/8' }, { subnet: '192.168.0.0/16' }],
            },
        ]

        component.networks = networks
        expect(component.isAnyIPv6SubnetVisible).toBeFalse()

        networks.push({
            subnets: [{ subnet: 'fe80::/64' }],
        })
        expect(component.isAnyIPv6SubnetVisible).toBeTrue()
    })

    it('should display proper utilization bars', async () => {
        component.loadNetworks({})
        await fixture.whenStable()
        component.loadNetworks({})
        await fixture.whenStable()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.networks.length).toBe(1)
        expect(component.networks[0].subnets.length).toBe(4)

        const barElements = fixture.debugElement.queryAll(By.directive(SubnetBarComponent))
        expect(barElements.length).toBe(4)

        for (let i = 0; i < barElements.length; i++) {
            const barElement = barElements[i]
            const bar: SubnetBarComponent = barElement.componentInstance
            expect(bar.isIPv6).toBeTrue()

            switch (i) {
                case 0:
                    expect(bar.hasZeroAddressStats).toBeFalse()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeFalse()
                    expect(bar.addrUtilization).toBe(10)
                    expect(bar.pdUtilization).toBe(15)
                    break
                case 1:
                    expect(bar.hasZeroAddressStats).toBeFalse()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeTrue()
                    expect(bar.addrUtilization).toBe(20)
                    expect(bar.pdUtilization).toBe(0)
                    break
                case 2:
                    expect(bar.hasZeroAddressStats).toBeTrue()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeFalse()
                    expect(bar.addrUtilization).toBe(0)
                    expect(bar.pdUtilization).toBe(35)
                    break
                case 3:
                    expect(bar.hasZeroAddressStats).toBeTrue()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeTrue()
                    expect(bar.addrUtilization).toBe(0)
                    expect(bar.pdUtilization).toBe(0)
                    break
            }
        }
    })

    it('should open and close tabs', fakeAsync(() => {
        component.openTabBySharedNetworkId(1)
        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        component.closeTabByIndex(1)

        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        component.closeTabByIndex(0)

        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
    }))
})
