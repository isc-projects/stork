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
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { ButtonModule } from 'primeng/button'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { TagModule } from 'primeng/tag'
import { CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                ButtonModule,
                ChartModule,
                CheckboxModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterTestingModule,
                TableModule,
                TagModule,
                TooltipModule,
                TreeModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                CascadedParametersBoardComponent,
                DelegatedPrefixBarComponent,
                DhcpOptionSetViewComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                NumberPipe,
                PlaceholderPipe,
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
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            cacheMaxAge: 1000,
                            options: [
                                {
                                    code: 1033,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            options: [
                                {
                                    code: 1034,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            options: [
                                {
                                    code: 1035,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                    },
                },
            ],
        }
        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 192.0.2.0/24 in shared network Fiber')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(5)

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

        expect(fieldsets[3].nativeElement.innerText).toContain('Cache Threshold')
        expect(fieldsets[3].nativeElement.innerText).toContain('0.25')
        expect(fieldsets[3].nativeElement.innerText).toContain('1000')

        // Ensure that the DHCP options are excluded from this list.
        expect(fieldsets[3].nativeElement.innerText).not.toContain('Options')
        expect(fieldsets[3].nativeElement.innerText).not.toContain('Options Hash')

        // DHCP options sit in their own fieldset.
        expect(fieldsets[4].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[4].nativeElement.innerText).toContain('1033')
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
        expect(fieldsets.length).toBe(5)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')
        expect(fieldsets[1].nativeElement.innerText).toContain('No pools configured.')

        expect(fieldsets[3].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No options configured.')
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
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::786',
                        },
                    ],
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
        expect(fieldsets.length).toBe(5)

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

        expect(fieldsets[3].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No options configured.')
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
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
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
        expect(fieldsets.length).toBe(5)

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

        expect(fieldsets[3].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display an IPv6 subnet with different fieldsets for different servers', () => {
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
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
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
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                },
                            ],
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                        },
                    },
                },
                {
                    id: 25432,
                    appName: 'bar@2001:db8:2::5',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
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
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                },
                            ],
                            optionsHash: '234',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                        },
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)

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

        expect(fieldsets[4].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[5].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[5].nativeElement.innerText).toContain('foo@2001:db8:1::1')
        expect(fieldsets[6].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[6].nativeElement.innerText).toContain('bar@2001:db8:2::5')
    })

    it('should return shared network attributes for IPv6 subnet', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            sharedNetworkId: 123,
            sharedNetwork: 'foo',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            id: 123,
            name: 'foo',
        })
    })

    it('should return shared network attributes for IPv4 subnet', () => {
        component.subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 234,
            sharedNetwork: 'bar',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            id: 234,
            name: 'bar',
        })
    })
})
