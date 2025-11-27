import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UtilizationBarComponent } from './utilization-bar.component'

describe('UtilizationBarComponent', () => {
    let component: UtilizationBarComponent
    let fixture: ComponentFixture<UtilizationBarComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(UtilizationBarComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should prepare a proper address utilization bar style', () => {
        component.utilizationPrimary = 30
        expect(component.utilizationStylePrimary.width).toBe('30%')
        component.utilizationPrimary = -10
        expect(component.utilizationStylePrimary.width).toBe('0%')
        component.utilizationPrimary = 110
        expect(component.utilizationStylePrimary.width).toBe('100%')
    })

    it('should prepare a proper delegated prefix utilization bar style', () => {
        component.utilizationSecondary = 60
        expect(component.utilizationStyleSecondary.width).toBe('60%')
        component.utilizationSecondary = -20
        expect(component.utilizationStyleSecondary.width).toBe('0%')
        component.utilizationSecondary = 120
        expect(component.utilizationStyleSecondary.width).toBe('100%')
    })

    it('should return a proper utilization bar modificator', () => {
        expect(component.getUtilizationBarModificatorClass(undefined)).toBe('utilization__bar--missing')
        expect(component.getUtilizationBarModificatorClass(null)).toBe('utilization__bar--missing')
        expect(component.getUtilizationBarModificatorClass(30)).toBe('utilization__bar--low')
        expect(component.getUtilizationBarModificatorClass(80)).toBe('utilization__bar--low')
        expect(component.getUtilizationBarModificatorClass(81)).toBe('utilization__bar--medium')
        expect(component.getUtilizationBarModificatorClass(90)).toBe('utilization__bar--medium')
        expect(component.getUtilizationBarModificatorClass(91)).toBe('utilization__bar--high')
        expect(component.getUtilizationBarModificatorClass(100)).toBe('utilization__bar--high')
        expect(component.getUtilizationBarModificatorClass(101)).toBe('utilization__bar--exceed')
    })

    it('has warning tooltip when utilization is greater than 100%', () => {
        component.utilizationPrimary = 101
        component.stats = {
            'total-nas': 100.0,
            'assigned-nas': 101.0,
            'declined-nas': 0,
            'total-pds': 200.0,
            'assigned-pds': 202.0,
        }

        expect(component.tooltip).toContain('101.0%')
        expect(component.tooltip).toContain('Data is unreliable')
    })

    it('has tooltip when utilization is known but the stats are not', () => {
        component.utilizationPrimary = 50

        expect(component.tooltip).toContain('50.0%')
        expect(component.tooltip).toContain('No statistics yet')
        expect(component.tooltip).toContain('Utilization')
    })

    it('has tooltip when utilization is unknown but the stats are known', () => {
        component.stats = {
            'total-nas': 100.0,
            'assigned-nas': 50.0,
            'declined-nas': 0,
            'total-pds': 200.0,
            'assigned-pds': 100.0,
        }
        expect(component.tooltip).not.toContain('No statistics yet')
        expect(component.tooltip).toContain('200')
        expect(component.tooltip).not.toContain('Utilization')
    })

    it('has tooltip when utilization and stats are unknown', () => {
        expect(component.tooltip).toBe('No statistics yet')
    })

    it('limits decimal places in utilization', () => {
        component.utilizationPrimary = 9.87654321

        expect(component.tooltip).toContain('9.9%')
        expect(component.tooltip).toContain('Utilization')
    })

    it('tooltip should be prepared for DHCPv4', () => {
        const stats = {
            'total-addresses': 4,
            'assigned-addresses': 2,
            'declined-addresses': 1,
        }

        component.stats = stats
        component.statsCollectedAt = '2022-12-28T14:59:00'
        component.utilizationPrimary = 5

        expect(component.tooltip).toContain('5')
        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6', () => {
        const stats = {
            'total-nas': 4,
            'assigned-nas': 2,
            'declined-nas': 1,
        }

        component.stats = stats
        component.statsCollectedAt = '2022-12-28T14:59:00'
        component.utilizationPrimary = 5
        component.utilizationSecondary = 6
        component.kindSecondary = 'PD'

        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
    })

    it('tooltip should be prepared for DHCPv6 with PDs', () => {
        const stats = {
            'total-nas': 4,
            'assigned-nas': 2,
            'declined-nas': 1,
            'total-pds': 6,
            'assigned-pds': 3,
        }

        component.stats = stats
        component.statsCollectedAt = '2022-12-28T14:59:00'

        expect(component.tooltip).toContain('4')
        expect(component.tooltip).toContain('2')
        expect(component.tooltip).toContain('1')
        expect(component.tooltip).toContain('6')
        expect(component.tooltip).toContain('3')
    })
})
