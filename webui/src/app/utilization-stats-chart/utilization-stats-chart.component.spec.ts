import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UtilizationStatsChartComponent } from './utilization-stats-chart.component'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'

describe('UtilizationStatsChartComponent', () => {
    let component: UtilizationStatsChartComponent
    let fixture: ComponentFixture<UtilizationStatsChartComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [provideNoopAnimations()],
        }).compileComponents()

        fixture = TestBed.createComponent(UtilizationStatsChartComponent)
        component = fixture.componentInstance
    })

    function setInputs(leaseType: 'na' | 'pd', network: object): void {
        fixture.componentRef.setInput('leaseType', leaseType)
        fixture.componentRef.setInput('network', network)
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should initialize all DHCPv4 address stats', () => {
        setInputs('na', {
            addrUtilization: 48.123,
            stats: {
                'total-addresses': 256,
                'assigned-addresses': 127,
                'declined-addresses': 11,
            },
        })

        expect(component.utilization).toBe(48.123)
        expect(component.total).toBe(BigInt(256))
        expect(component.assigned).toBe(BigInt(127))
        expect(component.declined).toBe(BigInt(11))

        expect(fixture.debugElement.nativeElement.textContent).toContain('Address Utilization (48%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(5)

        expect(stats[0].nativeElement.textContent).toContain('Total Addresses')
        expect(stats[0].nativeElement.textContent).toContain('256')
        expect(stats[1].nativeElement.textContent).toContain('Assigned Addresses')
        expect(stats[1].nativeElement.textContent).toContain('127')
        expect(stats[2].nativeElement.textContent).toContain('Free Addresses')
        expect(stats[2].nativeElement.textContent).toContain('129')
        expect(stats[3].nativeElement.textContent).toContain('Used Addresses')
        expect(stats[3].nativeElement.textContent).toContain('116')
        expect(stats[4].nativeElement.textContent).toContain('Declined Addresses')
        expect(stats[4].nativeElement.textContent).toContain('11')
    })

    it('should initialize all DHCPv6 address stats', () => {
        setInputs('na', {
            addrUtilization: 90,
            stats: {
                'total-nas': 6000,
                'assigned-nas': 2000,
                'declined-nas': 100,
            },
        })

        expect(component.utilization).toBe(90)
        expect(component.total).toBe(BigInt(6000))
        expect(component.assigned).toBe(BigInt(2000))
        expect(component.declined).toBe(BigInt(100))

        expect(fixture.debugElement.nativeElement.textContent).toContain('Address Utilization (90%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(5)

        expect(stats[0].nativeElement.textContent).toContain('Total Addresses')
        expect(stats[0].nativeElement.textContent).toContain('6.0k')
        expect(stats[1].nativeElement.textContent).toContain('Assigned Addresses')
        expect(stats[1].nativeElement.textContent).toContain('2.0k')
        expect(stats[2].nativeElement.textContent).toContain('Free Addresses')
        expect(stats[2].nativeElement.textContent).toContain('4.0k')
        expect(stats[3].nativeElement.textContent).toContain('Used Addresses')
        expect(stats[3].nativeElement.textContent).toContain('1.9k')
        expect(stats[4].nativeElement.textContent).toContain('Declined Addresses')
        expect(stats[4].nativeElement.textContent).toContain('100')
    })

    it('should initialize all prefix stats', () => {
        setInputs('pd', {
            pdUtilization: 49.8,
            stats: {
                'total-pds': 1000,
                'assigned-pds': 498,
            },
        })

        expect(component.hasStats).toBeTrue()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(49.8)
        expect(component.total).toBe(BigInt(1000))
        expect(component.assigned).toBe(BigInt(498))

        expect(fixture.debugElement.nativeElement.textContent).toContain('Prefix Utilization (50%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(3)

        expect(stats[0].nativeElement.textContent).toContain('Total Prefixes')
        expect(stats[0].nativeElement.textContent).toContain('1.0k')
        expect(stats[1].nativeElement.textContent).toContain('Assigned Prefixes')
        expect(stats[1].nativeElement.textContent).toContain('498')
        expect(stats[2].nativeElement.textContent).toContain('Free Prefixes')
        expect(stats[2].nativeElement.textContent).toContain('502')
    })

    it('should use address utilization when other stats are not available', () => {
        setInputs('na', {
            addrUtilization: 90,
            stats: {},
        })

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(90)
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()
        expect(component.declined).toBe(0n)

        expect(fixture.debugElement.nativeElement.textContent).toContain('Address Utilization (90%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })

    it('should use prefix utilization when other stats are not available', () => {
        setInputs('pd', {
            pdUtilization: 100,
            stats: {},
        })

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(100)
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()

        expect(fixture.debugElement.nativeElement.textContent).toContain('Prefix Utilization (100%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })

    it('should not fall over when no stats are available', () => {
        setInputs('pd', {
            stats: {},
        })

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeFalse()

        expect(component.utilization).toBeNull()
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()

        expect(fixture.debugElement.nativeElement.textContent).not.toContain('Prefix Utilization (0%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })

    it('should show uncertain addresses when there are no free addresses', () => {
        setInputs('na', {
            addrUtilization: 100,
            stats: {
                'total-addresses': 256,
                'assigned-addresses': 127,
                'declined-addresses': 240,
            },
        })

        expect(component.utilization).toBe(100)
        expect(component.total).toBe(BigInt(256))
        expect(component.assigned).toBe(BigInt(127))
        expect(component.declined).toBe(BigInt(240))

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(4)

        expect(stats[0].nativeElement.textContent).toContain('Total Addresses')
        expect(stats[0].nativeElement.textContent).toContain('256')
        expect(stats[1].nativeElement.textContent).toContain('Free Addresses')
        expect(stats[1].nativeElement.textContent).toContain('0')
        expect(stats[2].nativeElement.textContent).toContain('Uncertain Addresses')
        expect(stats[2].nativeElement.textContent).toContain('16')
        expect(stats[3].nativeElement.textContent).toContain('Declined Addresses')
        expect(stats[3].nativeElement.textContent).toContain('240')
    })

    it('should show uncertain addresses when there are free addresses', () => {
        setInputs('na', {
            addrUtilization: 100,
            stats: {
                'total-addresses': 512,
                'assigned-addresses': 4,
                'declined-addresses': 120,
            },
        })

        expect(component.utilization).toBe(100)
        expect(component.total).toBe(BigInt(512))
        expect(component.assigned).toBe(BigInt(4))
        expect(component.declined).toBe(BigInt(120))

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(4)

        expect(stats[0].nativeElement.textContent).toContain('Total Addresses')
        expect(stats[0].nativeElement.textContent).toContain('512')
        expect(stats[1].nativeElement.textContent).toContain('Free Addresses')
        expect(stats[1].nativeElement.textContent).toContain('388')
        expect(stats[2].nativeElement.textContent).toContain('Uncertain Addresses')
        expect(stats[2].nativeElement.textContent).toContain('4')
        expect(stats[3].nativeElement.textContent).toContain('Declined Addresses')
        expect(stats[3].nativeElement.textContent).toContain('120')
    })
})
