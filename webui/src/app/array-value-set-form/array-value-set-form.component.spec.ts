import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ArrayValueSetFormComponent } from './array-value-set-form.component'
import { FormControl, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { AutoComplete, AutoCompleteModule } from 'primeng/autocomplete'

describe('ArrayValueSetFormComponent', () => {
    let component: ArrayValueSetFormComponent<string>
    let fixture: ComponentFixture<ArrayValueSetFormComponent<string>>

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [ArrayValueSetFormComponent],
            imports: [FormsModule, NoopAnimationsModule, ReactiveFormsModule, AutoCompleteModule],
        })
        fixture = TestBed.createComponent(ArrayValueSetFormComponent<string>)
        component = fixture.componentInstance
        component.classFormControl = new FormControl<string>(null)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display AutoComplete component', () => {
        const autoComplete = fixture.debugElement.query(By.directive(AutoComplete))
        const acComponent = autoComplete.componentInstance as AutoComplete
        expect(acComponent).toBeTruthy()
    })
})
