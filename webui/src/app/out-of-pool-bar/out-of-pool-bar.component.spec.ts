import { ComponentFixture, TestBed } from '@angular/core/testing'

import { OutOfPoolBarComponent } from './out-of-pool-bar.component'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'

describe('OutOfPoolBarComponent', () => {
    let component: OutOfPoolBarComponent
    let fixture: ComponentFixture<OutOfPoolBarComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [OutOfPoolBarComponent, UtilizationBarComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(OutOfPoolBarComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should filter out-of-pool statistics correctly', () => {
        const statsIPv4 = {
            'total-addresses': 1000,
            'assigned-addresses': 500,
            'total-out-of-pool-addresses': 100,
            'assigned-out-of-pool-addresses': 50,
        }

        const statsIPv6 = {
            'total-pds': 1000,
            'assigned-pds': 500,
            'total-nas': 1000,
            'assigned-nas': 500,
            'total-out-of-pool-nas': 100,
            'assigned-out-of-pool-nas': 50,
            'total-out-of-pool-pds': 100,
            'assigned-out-of-pool-pds': 50,
        }

        component.stats = statsIPv4
        expect(component.stats).toEqual({
            'total-addresses': 100,
            'assigned-addresses': 50,
        })

        component.stats = statsIPv6
        expect(component.stats).toEqual({
            'total-nas': 100,
            'assigned-nas': 50,
        })

        component.isPD = true
        component.stats = statsIPv6
        expect(component.stats).toEqual({
            'total-pds': 100,
            'assigned-pds': 50,
        })
    })

    it('should determine if out-of-pool data is available', () => {
        // Valid IPv4 stats.
        component.utilization = 50
        component.stats = {
            'total-out-of-pool-addresses': 100,
        }
        expect(component.hasOutOfPoolData).toBeTrue()

        // Valid IPv6 stats.
        component.utilization = 50
        component.stats = {
            'total-out-of-pool-nas': 100,
        }
        expect(component.hasOutOfPoolData).toBeTrue()

        // Valid PD stats.
        component.isPD = true
        component.utilization = 50
        component.stats = {
            'total-out-of-pool-pds': 100,
        }
        expect(component.hasOutOfPoolData).toBeTrue()

        // Missing stats.
        component.isPD = false
        component.utilization = 50
        component.stats = null
        expect(component.hasOutOfPoolData).toBeFalse()

        // Missing utilization.
        component.utilization = null
        component.stats = {
            'total-out-of-pool-addresses': 100,
        }
        expect(component.hasOutOfPoolData).toBeTrue()

        // Zero total statistics.
        component.utilization = 50
        component.stats = {
            'total-out-of-pool-addresses': 0,
        }
        expect(component.hasOutOfPoolData).toBeFalse()

        component.utilization = 50
        component.stats = {
            'total-out-of-pool-nas': 0,
        }
        expect(component.hasOutOfPoolData).toBeFalse()

        component.utilization = 50
        component.stats = {
            'total-out-of-pool-pds': 0,
        }
        expect(component.hasOutOfPoolData).toBeFalse()
    })
})
