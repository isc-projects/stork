import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TriStateCheckboxComponent } from './tri-state-checkbox.component'
import { By } from '@angular/platform-browser'
import { Checkbox } from 'primeng/checkbox'

describe('TriStateCheckboxComponent', () => {
    let component: TriStateCheckboxComponent
    let fixture: ComponentFixture<TriStateCheckboxComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TriStateCheckboxComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(TriStateCheckboxComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should handle three states', () => {
        const pCheckbox = fixture.debugElement.query(By.directive(Checkbox))
        expect(pCheckbox).toBeTruthy()
        expect(component.value()).toBeNull()

        pCheckbox.nativeElement.click()
        fixture.detectChanges()
        expect(component.value()).toBeTrue()

        pCheckbox.nativeElement.click()
        fixture.detectChanges()
        expect(component.value()).toBeFalse()

        pCheckbox.nativeElement.click()
        fixture.detectChanges()
        expect(component.value()).toBeNull()
    })

    it('should have label', () => {
        fixture.componentRef.setInput('label', 'test')
        fixture.componentRef.setInput('inputID', 'test-id')
        fixture.detectChanges()

        const label = fixture.debugElement.query(By.css('label'))
        expect(label).toBeTruthy()
        expect(label.nativeElement.textContent).toBe('test')
    })

    it('should not have label', () => {
        let label = fixture.debugElement.query(By.css('label'))
        expect(label).toBeFalsy()

        fixture.componentRef.setInput('label', 'test')
        fixture.detectChanges()

        label = fixture.debugElement.query(By.css('label'))
        expect(label).toBeFalsy()

        fixture.componentRef.setInput('label', undefined)
        fixture.componentRef.setInput('inputID', 'test-id')
        fixture.detectChanges()

        label = fixture.debugElement.query(By.css('label'))
        expect(label).toBeFalsy()
    })
})
