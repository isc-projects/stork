import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UtilizationStatsChartComponent } from './utilization-stats-chart.component'
import { By } from '@angular/platform-browser'
import { ChartModule } from 'primeng/chart'
import { HumanCountComponent } from '../human-count/human-count.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { TooltipModule } from 'primeng/tooltip'

describe('UtilizationStatsChartComponent', () => {
    let component: UtilizationStatsChartComponent
    let fixture: ComponentFixture<UtilizationStatsChartComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [ChartModule, NoopAnimationsModule, TooltipModule],
            declarations: [HumanCountComponent, HumanCountPipe, LocalNumberPipe, UtilizationStatsChartComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(UtilizationStatsChartComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should initialize all DHCPv4 address stats', () => {
        component.leaseType = 'na'
        component.network = {
            addrUtilization: 48.123,
            stats: {
                'total-addresses': 256,
                'assigned-addresses': 128,
                'declined-addresses': 11,
            },
        }

        fixture.detectChanges()

        expect(component.utilization).toBe(48.123)
        expect(component.total).toBe(BigInt(256))
        expect(component.assigned).toBe(BigInt(128))
        expect(component.declined).toBe(BigInt(11))

        expect(fixture.debugElement.nativeElement.innerText).toContain('Address Utilization (48%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(4)

        expect(stats[0].nativeElement.innerText).toContain('Total Addresses')
        expect(stats[0].nativeElement.innerText).toContain('256')
        expect(stats[1].nativeElement.innerText).toContain('Assigned Addresses')
        expect(stats[1].nativeElement.innerText).toContain('128')
        expect(stats[2].nativeElement.innerText).toContain('Used Addresses')
        expect(stats[2].nativeElement.innerText).toContain('117')
        expect(stats[3].nativeElement.innerText).toContain('Declined Addresses')
        expect(stats[3].nativeElement.innerText).toContain('11')
    })

    it('should initialize all DHCPv6 address stats', () => {
        component.leaseType = 'na'
        component.network = {
            addrUtilization: 90,
            stats: {
                'total-nas': 6000,
                'assigned-nas': 2000,
                'declined-nas': 100,
            },
        }

        fixture.detectChanges()

        expect(component.utilization).toBe(90)
        expect(component.total).toBe(BigInt(6000))
        expect(component.assigned).toBe(BigInt(2000))
        expect(component.declined).toBe(BigInt(100))

        expect(fixture.debugElement.nativeElement.innerText).toContain('Address Utilization (90%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(4)

        expect(stats[0].nativeElement.innerText).toContain('Total Addresses')
        expect(stats[0].nativeElement.innerText).toContain('6.0k')
        expect(stats[1].nativeElement.innerText).toContain('Assigned Addresses')
        expect(stats[1].nativeElement.innerText).toContain('2.0k')
        expect(stats[2].nativeElement.innerText).toContain('Used Addresses')
        expect(stats[2].nativeElement.innerText).toContain('1.9k')
        expect(stats[3].nativeElement.innerText).toContain('Declined Addresses')
        expect(stats[3].nativeElement.innerText).toContain('100')
    })

    it('should initialize all prefix stats', () => {
        component.leaseType = 'pd'
        component.network = {
            pdUtilization: 49.8,
            stats: {
                'total-pds': 1000,
                'assigned-pds': 498,
            },
        }

        fixture.detectChanges()

        expect(component.hasStats).toBeTrue()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(49.8)
        expect(component.total).toBe(BigInt(1000))
        expect(component.assigned).toBe(BigInt(498))

        expect(fixture.debugElement.nativeElement.innerText).toContain('Prefix Utilization (50%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(2)

        expect(stats[0].nativeElement.innerText).toContain('Total Prefixes')
        expect(stats[0].nativeElement.innerText).toContain('1.0k')
        expect(stats[1].nativeElement.innerText).toContain('Assigned Prefixes')
        expect(stats[1].nativeElement.innerText).toContain('498')
    })

    it('should use address utilization when other stats are not available', () => {
        component.leaseType = 'na'
        component.network = {
            addrUtilization: 90,
            stats: {},
        }

        fixture.detectChanges()

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(90)
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()
        expect(component.declined).toBe(0n)

        expect(fixture.debugElement.nativeElement.innerText).toContain('Address Utilization (90%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })

    it('should use prefix utilization when other stats are not available', () => {
        component.leaseType = 'pd'
        component.network = {
            pdUtilization: 100,
            stats: {},
        }

        fixture.detectChanges()

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeTrue()

        expect(component.utilization).toBe(100)
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()

        expect(fixture.debugElement.nativeElement.innerText).toContain('Prefix Utilization (100%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })

    it('should not fall over when no stats are available', () => {
        component.leaseType = 'pd'
        component.network = {
            stats: {},
        }

        fixture.detectChanges()

        expect(component.hasStats).toBeFalse()
        expect(component.hasUtilization).toBeFalse()

        expect(component.utilization).toBeNull()
        expect(component.total).toBeNull()
        expect(component.assigned).toBeNull()

        expect(fixture.debugElement.nativeElement.innerText).not.toContain('Prefix Utilization (0%)')

        const stats = fixture.debugElement.queryAll(By.css('tr'))
        expect(stats.length).toBe(0)
    })
})
