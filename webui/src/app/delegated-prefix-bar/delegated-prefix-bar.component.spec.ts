import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DelegatedPrefixBarComponent } from './delegated-prefix-bar.component'

describe('DelegatedPrefixBarComponent', () => {
    let component: DelegatedPrefixBarComponent
    let fixture: ComponentFixture<DelegatedPrefixBarComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(DelegatedPrefixBarComponent)
        component = fixture.componentInstance

    })

    /** Renders the component with the given prefix pool. */
    function render(pool: { prefix: string; delegatedLength: number; excludedPrefix?: string }): void {
        component.pool = pool
        fixture.detectChanges()
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display prefix and length', () => {
        render({ prefix: 'fe80::/64', delegatedLength: 80 })
        expect(
            (fixture.debugElement.nativeElement as HTMLElement).textContent
                .trim()
                // Replace &nbsp character.
                .replace(/\u00a0/g, ' ')
        ).toBe('fe80::/64 del.: 80')
    })

    it('should shorten the excluded prefix', () => {
        render({ prefix: 'fe80::/64', delegatedLength: 80, excludedPrefix: 'fe80:42::/96' })
        expect(component.shortExcludedPrefix).toBe('~:42::/96')
    })

    it('should not shorten if the excluded prefix has no common part with a prefix', () => {
        render({ prefix: 'fe80::/64', delegatedLength: 80, excludedPrefix: '3001::/96' })
        expect(component.shortExcludedPrefix).toBe('3001::/96')
    })

    it('should handle an error on the invalid excluded prefix', () => {
        render({ prefix: 'fe80::/64', delegatedLength: 80, excludedPrefix: 'foo' })
        expect(component.shortExcludedPrefix).toBe('foo')
    })

    it('should display an excluded prefix', () => {
        render({ prefix: 'fe80::/64', delegatedLength: 80, excludedPrefix: 'fe80:42::/96' })
        expect(
            (fixture.debugElement.nativeElement as HTMLElement).textContent
                .trim()
                // Replace &nbsp character.
                .replace(/\u00a0/g, ' ')
        ).toBe('fe80::/64 del.: 80 ex.: ~:42::/96')
    })
})
