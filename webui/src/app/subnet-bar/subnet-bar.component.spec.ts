import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetBarComponent } from './subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { By } from '@angular/platform-browser'
import { DebugElement } from '@angular/core'

describe('SubnetBarComponent', () => {
    let component: SubnetBarComponent
    let fixture: ComponentFixture<SubnetBarComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [TooltipModule],
                declarations: [SubnetBarComponent],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(SubnetBarComponent)
        component = fixture.componentInstance
        component.subnet = {
            localSubnets: [
                {
                    stats: null,
                },
            ],
        }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('tooltip should be prepared for DHCPv4', () => {
        const subnet4 = {
            subnet: '192.168.0.0/24',
            localSubnets: [
                {
                    stats: {
                        'total-addresses': 4,
                        'assigned-addresses': 2,
                        'declined-addresses': 1,
                    },
                },
            ],
        }

        component.subnet = subnet4

        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            localSubnets: [
                {
                    stats: {
                        'total-nas': 4,
                        'assigned-nas': 2,
                        'declined-nas': 1,
                    },
                },
            ],
        }

        component.subnet = subnet6

        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6 with PDs', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            localSubnets: [
                {
                    stats: {
                        'total-nas': 4,
                        'assigned-nas': 2,
                        'declined-nas': 1,
                        'total-pds': 6,
                        'assigned-pds': 3,
                    },
                },
            ],
        }

        component.subnet = subnet6

        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('3')
    }),
        it('subnet bar cannot extend beyond the container', async () => {
            // Returns an IPv6 subnet mock with given utilization. The utilization
            // should be a ratio (from 0% to 100%) of assigned to total NAs/PDs.
            function getSubnet(utilization: number) {
                return {
                    addrUtilization: utilization,
                    subnet: '3000::0/24',
                    localSubnets: [
                        {
                            stats: {
                                'total-nas': 100.0,
                                'assigned-nas': (100.0 * utilization) / 100.0,
                                'declined-nas': 0,
                                'total-pds': 200.0,
                                'assigned-pds': (200.0 * utilization) / 100.0,
                            },
                        },
                    ],
                }
            }

            // Check if the bar extends beyond the container.
            function extendBeyond(): boolean {
                const parent = fixture.debugElement.query(By.css('.utilization'))
                const parentElement = parent.nativeElement as Element
                const parentRect = parentElement.getBoundingClientRect()
                const bar = fixture.debugElement.query(By.css('.bar'))
                const barElement = bar.nativeElement as Element
                const barRect = barElement.getBoundingClientRect()

                if (
                    barRect.top < parentRect.top ||
                    barRect.bottom > parentRect.bottom ||
                    barRect.left < parentRect.left ||
                    barRect.right > parentRect.right
                ) {
                    return true
                }

                return false
            }

            // Utilization below 100%, usual situation.
            component.subnet = getSubnet(50)
            fixture.detectChanges()
            expect(extendBeyond()).toBeFalse()

            // Utilization equals to 100%, the subnet bar should
            // have a maximal width as allowed by the container.
            component.subnet = getSubnet(100)
            fixture.detectChanges()
            expect(extendBeyond()).toBeFalse()

            // Utilization above 100%, unusual case, but it shouldn't
            // cause UI glitches.
            component.subnet = getSubnet(150)
            fixture.detectChanges()
            expect(extendBeyond()).toBeFalse()

            // Utilization below 0%, invalid or buggy utilization.
            // Anyway, UI shouldn't be broken.
            component.subnet = getSubnet(-50)
            fixture.detectChanges()
            expect(extendBeyond()).toBeFalse()
        })
})
