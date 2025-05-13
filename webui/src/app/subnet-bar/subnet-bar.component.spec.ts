import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetBarComponent } from './subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { By } from '@angular/platform-browser'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { provideRouter, RouterModule } from '@angular/router'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'

describe('SubnetBarComponent', () => {
    let component: SubnetBarComponent
    let fixture: ComponentFixture<SubnetBarComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [TooltipModule, RouterModule],
            declarations: [SubnetBarComponent, EntityLinkComponent, UtilizationBarComponent],
            providers: [provideRouter([])],
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
})
