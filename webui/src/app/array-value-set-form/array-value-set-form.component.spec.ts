import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ArrayValueSetFormComponent } from './array-value-set-form.component'
import { Chips, ChipsModule } from 'primeng/chips'
import { FormControl, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'

describe('ArrayValueSetFormComponent', () => {
    let component: ArrayValueSetFormComponent<string>
    let fixture: ComponentFixture<ArrayValueSetFormComponent<string>>

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [ArrayValueSetFormComponent],
            imports: [ChipsModule, FormsModule, NoopAnimationsModule, ReactiveFormsModule],
        })
        fixture = TestBed.createComponent(ArrayValueSetFormComponent<string>)
        component = fixture.componentInstance
        component.classFormControl = new FormControl<string>(null)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display chips component', () => {
        const chips = fixture.debugElement.query(By.directive(Chips))
        const chipsComponent = chips.componentInstance as Chips
        expect(chipsComponent).toBeTruthy()
    })
})
