import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DelegatedPrefixBarComponent } from './delegated-prefix-bar.component'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'
import { TooltipModule } from 'primeng/tooltip'

describe('DelegatedPrefixBarComponent', () => {
    let component: DelegatedPrefixBarComponent
    let fixture: ComponentFixture<DelegatedPrefixBarComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TooltipModule],
            declarations: [DelegatedPrefixBarComponent, UtilizationBarComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(DelegatedPrefixBarComponent)
        component = fixture.componentInstance

        component.pool = {
            prefix: 'fe80::/64',
            delegatedLength: 80,
        }

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display prefix and length', () => {
        expect(
            (fixture.debugElement.nativeElement as HTMLElement).textContent
                .trim()
                // Replace &nbsp character.
                .replace(/\u00a0/g, ' ')
        ).toBe('fe80::/64 del.: 80')
    })

    it('should shorten the excluded prefix', () => {
        component.pool.excludedPrefix = 'fe80:42::/96'
        expect(component.shortExcludedPrefix).toBe('~:42::/96')
    })

    it('should not shorten if the excluded prefix has no common part with a prefix', () => {
        component.pool.excludedPrefix = '3001::/96'
        expect(component.shortExcludedPrefix).toBe('3001::/96')
    })

    it('should handle an error on the invalid excluded prefix', () => {
        component.pool.excludedPrefix = 'foo'
        expect(component.shortExcludedPrefix).toBe('foo')
    })

    it('should display an excluded prefix', () => {
        component.pool.excludedPrefix = 'fe80:42::/96'
        fixture.detectChanges()
        expect(
            (fixture.debugElement.nativeElement as HTMLElement).textContent
                .trim()
                // Replace &nbsp character.
                .replace(/\u00a0/g, ' ')
        ).toBe('fe80::/64 del.: 80 ex.: ~:42::/96')
    })
})
