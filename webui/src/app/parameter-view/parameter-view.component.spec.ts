import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ParameterViewComponent } from './parameter-view.component'
import { UncamelPipe } from '../pipes/uncamel.pipe'
import { UnhyphenPipe } from '../pipes/unhyphen.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'

describe('ParameterViewComponent', () => {
    let component: ParameterViewComponent
    let fixture: ComponentFixture<ParameterViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [ParameterViewComponent, PlaceholderPipe, UncamelPipe, UnhyphenPipe],
        }).compileComponents()

        fixture = TestBed.createComponent(ParameterViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display object', () => {
        component.parameter = {
            torque: 50,
            'power-output': 'regular',
            isNormal: true,
            horsePower: '',
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Torque: 50')
        expect(fixture.nativeElement.innerText).toContain('Power Output: regular')
        expect(fixture.nativeElement.innerText).toContain('Is Normal: true')
        expect(fixture.nativeElement.innerText).toContain('Horse Power: (empty)')
    })

    it('should display array', () => {
        component.parameter = [50, 'regular', true, '']
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('50')
        expect(fixture.nativeElement.innerText).toContain('regular')
        expect(fixture.nativeElement.innerText).toContain('true')
        expect(fixture.nativeElement.innerText).toContain('(empty)')
    })

    it('should display basic type value', () => {
        component.parameter = 'screw'
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('screw')
    })

    it('should display placeholder for an empty object', () => {
        component.parameter = {}
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('(not specified)')
    })

    it('should display placeholder for an empty array', () => {
        component.parameter = []
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('(empty)')
    })
})
