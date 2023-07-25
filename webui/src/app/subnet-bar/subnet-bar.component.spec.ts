import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetBarComponent } from './subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { By } from '@angular/platform-browser'
import { RouterTestingModule } from '@angular/router/testing'
import { EntityLinkComponent } from '../entity-link/entity-link.component'

describe('SubnetBarComponent', () => {
    let component: SubnetBarComponent
    let fixture: ComponentFixture<SubnetBarComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [TooltipModule, RouterTestingModule],
            declarations: [SubnetBarComponent, EntityLinkComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SubnetBarComponent)
        component = fixture.componentInstance
        component.subnet = {
            subnet: '',
            stats: null,
        }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('tooltip should be prepared for DHCPv4', () => {
        const subnet4 = {
            subnet: '192.168.0.0/24',
            stats: {
                'total-addresses': 4,
                'assigned-addresses': 2,
                'declined-addresses': 1,
            },
            addrUtilization: 5,
        }

        component.subnet = subnet4

        expect(component.tooltip).toContain('5')
        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            stats: {
                'total-nas': 4,
                'assigned-nas': 2,
                'declined-nas': 1,
            },
            addrUtilization: 5,
            pdUtilization: 6,
        }

        component.subnet = subnet6

        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6 with PDs', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            stats: {
                'total-nas': 4,
                'assigned-nas': 2,
                'declined-nas': 1,
                'total-pds': 6,
                'assigned-pds': 3,
            },
        }

        component.subnet = subnet6

        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('3')
    })

    it('subnet bar cannot extend beyond the container', async () => {
        /**
         * Returns an IPv6 subnet mock with given utilization. The utilization
         * should be a ratio (from 0% to 100%) of assigned to total NAs/PDs.
         */
        function getSubnet(utilization: number) {
            return {
                addrUtilization: utilization,
                subnet: '3000::0/24',
                stats: {
                    'total-nas': 100.0,
                    'assigned-nas': (100.0 * utilization) / 100.0,
                    'declined-nas': 0,
                    'total-pds': 200.0,
                    'assigned-pds': (200.0 * utilization) / 100.0,
                },
            }
        }

        /** Check if the bar extends beyond the container. */
        function extendBeyond(): boolean {
            const parent = fixture.debugElement.query(By.css('.utilization'))
            const parentElement = parent.nativeElement as Element
            const parentRect = parentElement.getBoundingClientRect()
            const bar = fixture.debugElement.query(By.css('.utilization__bar'))
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

    it('has warning tooltip when utilization is greater than 100%', () => {
        component.subnet = {
            addrUtilization: 101,
            subnet: '3000::0/24',
            stats: {
                'total-nas': 100.0,
                'assigned-nas': 101.0,
                'declined-nas': 0,
                'total-pds': 200.0,
                'assigned-pds': 202.0,
            },
        }

        fixture.detectChanges()
        expect(component.tooltip).toContain('Data is unreliable')
    })

    it('returns the address utilization as a number', () => {
        component.subnet.addrUtilization = 42
        expect(component.addrUtilization).toBe(42)
        component.subnet.addrUtilization = null
        expect(component.addrUtilization).toBe(0)
    })

    it('returns the delegated prefix utilization as a number', () => {
        component.subnet.pdUtilization = 42
        expect(component.pdUtilization).toBe(42)
        component.subnet.pdUtilization = null
        expect(component.pdUtilization).toBe(0)
    })

    it('should detect IPv6 subnets', () => {
        component.subnet.subnet = 'fe80::/64'
        expect(component.isIPv6).toBeTrue()
        component.subnet.subnet = '10.0.0.0/8'
        expect(component.isIPv6).toBeFalse()
    })

    it('should prepare a proper address utilization bar style', () => {
        component.subnet.addrUtilization = 30
        expect(component.addrUtilizationStyle.width).toBe('30%')
        component.subnet.addrUtilization = -10
        expect(component.addrUtilizationStyle.width).toBe('0%')
        component.subnet.addrUtilization = 110
        expect(component.addrUtilizationStyle.width).toBe('100%')
    })

    it('should prepare a proper delegated prefix utilization bar style', () => {
        component.subnet.pdUtilization = 60
        expect(component.pdUtilizationStyle.width).toBe('60%')
        component.subnet.pdUtilization = -20
        expect(component.pdUtilizationStyle.width).toBe('0%')
        component.subnet.pdUtilization = 120
        expect(component.pdUtilizationStyle.width).toBe('100%')
    })

    it('should return a proper utilization bar modificator', () => {
        expect(component.getUtilizationBarModificatorClass(30)).toBe('utilization__bar--missing')
        expect(component.getUtilizationBarModificatorClass(85)).toBe('utilization__bar--missing')
        expect(component.getUtilizationBarModificatorClass(95)).toBe('utilization__bar--missing')
        expect(component.getUtilizationBarModificatorClass(195)).toBe('utilization__bar--missing')

        component.subnet.stats = {}

        expect(component.getUtilizationBarModificatorClass(30)).toBe('utilization__bar--low')
        expect(component.getUtilizationBarModificatorClass(80)).toBe('utilization__bar--low')
        expect(component.getUtilizationBarModificatorClass(81)).toBe('utilization__bar--medium')
        expect(component.getUtilizationBarModificatorClass(90)).toBe('utilization__bar--medium')
        expect(component.getUtilizationBarModificatorClass(91)).toBe('utilization__bar--high')
        expect(component.getUtilizationBarModificatorClass(100)).toBe('utilization__bar--high')
        expect(component.getUtilizationBarModificatorClass(101)).toBe('utilization__bar--exceed')
    })

    it('should display single bar for IPv4', () => {
        component.subnet.subnet = '10.0.0.0/8'
        fixture.detectChanges()
        const elements = fixture.debugElement.queryAll(By.css('.utilization__bar'))
        expect(elements.length).toBe(1)
    })

    it('should display double bar for IPv6', () => {
        component.subnet.subnet = 'fe80::/64'
        fixture.detectChanges()
        const elements = fixture.debugElement.queryAll(By.css('.utilization__bar'))
        expect(elements.length).toBe(2)
    })

    it('should not indicate that there are the NA or PD statistics are zero if the statistics are not fetched yet', () => {
        component.subnet.stats = null
        expect(component.hasZeroAddressStats).toBeFalse()
        expect(component.hasZeroDelegatedPrefixStats).toBeFalse()
    })

    it('should detect properly that there are no PD statistics', () => {
        component.subnet.stats = {}
        expect(component.hasZeroAddressStats).toBeTrue()
        component.subnet.stats['total-pds'] = 0
        expect(component.hasZeroAddressStats).toBeTrue()
    })

    it('should detect properly that there are no NA statistics', () => {
        component.subnet.stats = {}
        expect(component.hasZeroAddressStats).toBeTrue()
        component.subnet.stats['total-nas'] = 0
        expect(component.hasZeroAddressStats).toBeTrue()
    })
})
