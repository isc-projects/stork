import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UtilizationStatsChartsComponent } from './utilization-stats-charts.component'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'

describe('UtilizationStatsChartsComponent', () => {
    let component: UtilizationStatsChartsComponent
    let fixture: ComponentFixture<UtilizationStatsChartsComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [provideNoopAnimations()],
        }).compileComponents()

        fixture = TestBed.createComponent(UtilizationStatsChartsComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display total statistics only', () => {
        component.network = {
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
                    daemonLabel: 'DHCPv4@localhost',
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
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(1)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('240')
        expect(charts[0].nativeElement.innerText).toContain('70')
        expect(charts[0].nativeElement.innerText).toContain('10')
    })

    it('should display address statistics for more servers', () => {
        component.network = {
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
                    daemonLabel: 'DHCPv4@localhost',
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 50,
                        'declined-addresses': 4,
                    },
                },
                {
                    id: 12223,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 20,
                        'declined-addresses': 6,
                    },
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(3)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('240')
        expect(charts[0].nativeElement.innerText).toContain('70')
        expect(charts[0].nativeElement.innerText).toContain('10')

        expect(charts[1].nativeElement.innerText).toContain('DHCPv4@localhost')
        expect(charts[1].nativeElement.innerText).toContain('240')
        expect(charts[1].nativeElement.innerText).toContain('50')
        expect(charts[1].nativeElement.innerText).toContain('4')

        expect(charts[2].nativeElement.innerText).toContain('DHCPv6@localhost')
        expect(charts[2].nativeElement.innerText).toContain('240')
        expect(charts[2].nativeElement.innerText).toContain('20')
        expect(charts[2].nativeElement.innerText).toContain('6')
    })

    it('should display address statistics for no pools when utilization exists', () => {
        component.network = {
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 120,
                'declined-addresses': 0,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonLabel: 'DHCPv4@localhost',
                },
                {
                    id: 12223,
                    daemonLabel: 'DHCPv6@localhost',
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(1)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('240')
        expect(charts[0].nativeElement.innerText).toContain('120')
        expect(charts[0].nativeElement.innerText).toContain('0')
    })

    it('should display total prefix statistics only', () => {
        component.network = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 0,
            pdUtilization: 60,
            stats: {
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonLabel: 'DHCPv4@localhost',
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-pds': 500,
                        'assigned-pds': 358,
                    },
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(1)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('500')
        expect(charts[0].nativeElement.innerText).toContain('358')
    })

    it('should display total prefix statistics for more servers', () => {
        component.network = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 0,
            pdUtilization: 60,
            stats: {
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonLabel: 'DHCPv4@localhost',
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-pds': 300,
                        'assigned-pds': 200,
                    },
                },
                {
                    id: 12223,
                    daemonLabel: 'DHCPv6@localhost',
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-pds': 200,
                        'assigned-pds': 158,
                    },
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(3)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('500')
        expect(charts[0].nativeElement.innerText).toContain('358')

        expect(charts[1].nativeElement.innerText).toContain('DHCPv4@localhost')
        expect(charts[1].nativeElement.innerText).toContain('300')
        expect(charts[1].nativeElement.innerText).toContain('200')

        expect(charts[2].nativeElement.innerText).toContain('DHCPv6@localhost')
        expect(charts[2].nativeElement.innerText).toContain('200')
        expect(charts[2].nativeElement.innerText).toContain('158')
    })

    it('should display prefix statistics for no pools when utilization exists', () => {
        component.network = {
            subnet: '2001:db8:1::/64',
            addrUtilization: 0,
            pdUtilization: 60,
            stats: {
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonLabel: 'DHCPv6@localhost',
                    stats: {
                        'total-pds': 500,
                        'assigned-pds': 358,
                    },
                },
            ],
        }
        fixture.detectChanges()

        const charts = fixture.debugElement.queryAll(By.css('app-utilization-stats-chart'))
        expect(charts.length).toBe(1)

        expect(charts[0].nativeElement.innerText).toContain('Total')
        expect(charts[0].nativeElement.innerText).toContain('500')
        expect(charts[0].nativeElement.innerText).toContain('358')
    })
})
