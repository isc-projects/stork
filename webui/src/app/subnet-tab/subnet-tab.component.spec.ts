import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ChartModule } from 'primeng/chart'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { NumberPipe } from '../pipes/number.pipe'
import { FieldsetModule } from 'primeng/fieldset'
import { DividerModule } from 'primeng/divider'
import { TableModule } from 'primeng/table'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { RouterTestingModule } from '@angular/router/testing'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { SubnetTabComponent } from './subnet-tab.component'
import { By } from '@angular/platform-browser'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                ChartModule,
                DividerModule,
                FieldsetModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterTestingModule,
                TableModule,
                TooltipModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                DelegatedPrefixBarComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                NumberPipe,
                SubnetTabComponent,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display an IPv4 subnet', () => {
        component.subnet = {
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    appName: 'foo@192.0.2.1',
                    pools: ['192.0.2.1-192.0.2.100'],
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 192.0.2.0/24 in shared network Fiber')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('192.0.2.1-192.0.2.100')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(1)
    })

    it('should display an IPv4 subnet without pools', () => {
        component.subnet = {
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    appName: 'foo@192.0.2.1',
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 192.0.2.0/24 in shared network Fiber')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(2)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')
        expect(fieldsets[1].nativeElement.innerText).toContain('No pools configured.')
    })

    it('should display an IPv6 subnet', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 60,
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    appName: 'foo@2001:db8:1::1',
                    pools: ['2001:db8:1::2-2001:db8:1::786'],
                    stats: {
                        'total-nas': 1000,
                        'assigned-nas': 30,
                        'declined-nas': 10,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@2001:db8:1::1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::786')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(1)
    })

    it('should display an IPv6 subnet with address pools and prefixes', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    appName: 'foo@2001:db8:1::1',
                    pools: ['2001:db8:1::2-2001:db8:1::768'],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 980,
                        'declined-nas': 10,
                        'total-pds': 500,
                        'assigned-pds': 358,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@2001:db8:1::1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        const prefixBar = fieldsets[1].query(By.css('app-delegated-prefix-bar'))
        expect(prefixBar).toBeTruthy()
        expect(prefixBar.nativeElement.innerText).toContain('3000::')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(2)
    })

    it('should display an IPv6 subnet with different pools for different servers', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    appName: 'foo@2001:db8:1::1',
                    pools: ['2001:db8:1::2-2001:db8:1::768'],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 500,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 200,
                    },
                },
                {
                    id: 25432,
                    appName: 'bar@2001:db8:2::5',
                    pools: ['2001:db8:1::2-2001:db8:1::768'],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                        {
                            prefix: '3000:1::',
                            delegatedLength: 96,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 480,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 158,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(4)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@2001:db8:1::1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')
        expect(fieldsets[0].nativeElement.innerText).toContain('bar@2001:db8:2::5')
        expect(fieldsets[0].nativeElement.innerText).toContain('25432')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('foo@2001:db8:1::1')

        let poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        let prefixBars = fieldsets[1].queryAll(By.css('app-delegated-prefix-bar'))
        expect(prefixBars.length).toBe(1)

        expect(fieldsets[2].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[2].nativeElement.innerText).toContain('bar@2001:db8:2::5')

        poolBar = fieldsets[2].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        prefixBars = fieldsets[2].queryAll(By.css('app-delegated-prefix-bar'))
        expect(prefixBars.length).toBe(2)
        expect(prefixBars[0].nativeElement.innerText).toContain('3000::')
        expect(prefixBars[1].nativeElement.innerText).toContain('3000:1::')

        const charts = fieldsets[3].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(6)
    })

    it('should return shared network attributes for IPv6 subnet', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            sharedNetwork: 'foo',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            text: 'foo',
            dhcpVersion: 6,
        })
    })

    it('should return shared network attributes for IPv4 subnet', () => {
        component.subnet = {
            subnet: '192.0.2.0/24',
            sharedNetwork: 'bar',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            text: 'bar',
            dhcpVersion: 4,
        })
    })
})
