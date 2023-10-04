import { ComponentFixture, TestBed } from '@angular/core/testing'

import { HumanCountComponent } from './human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { LocalNumberPipe } from '../pipes/local-number.pipe'

describe('HumanCountComponent', () => {
    let component: HumanCountComponent
    let fixture: ComponentFixture<HumanCountComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TooltipModule],
            declarations: [HumanCountComponent, HumanCountPipe, LocalNumberPipe],
        }).compileComponents()

        fixture = TestBed.createComponent(HumanCountComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert the string value', () => {
        component.value = '42'
        expect(component.value as any).toBe(BigInt(42))
    })

    it('should recognize a value', () => {
        component.value = 42
        expect(component.hasValue).toBeTrue()
        component.value = null
        expect(component.hasValue).toBeFalse()
        component.value = 'foo'
        expect(component.hasValue).toBeTrue()
    })

    it('should recognize a valid value', () => {
        component.value = 42
        expect(component.hasValidValue).toBeTrue()
        component.value = BigInt(42)
        expect(component.hasValidValue).toBeTrue()
        component.value = null
        expect(component.hasValidValue).toBeFalse()
        component.value = 'foo'
        expect(component.hasValidValue).toBeFalse()
    })

    it('should recognize an invalid value', () => {
        component.value = 42
        expect(component.hasInvalidValue).toBeFalse()
        component.value = BigInt(42)
        expect(component.hasInvalidValue).toBeFalse()
        component.value = null
        expect(component.hasInvalidValue).toBeFalse()
        component.value = 'foo'
        expect(component.hasInvalidValue).toBeTrue()
    })
})
