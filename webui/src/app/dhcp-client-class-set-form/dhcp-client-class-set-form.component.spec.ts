import { ComponentFixture, TestBed } from '@angular/core/testing'
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { By } from '@angular/platform-browser'
import { FloatLabelModule } from 'primeng/floatlabel'
import { AutoComplete, AutoCompleteModule } from 'primeng/autocomplete'

describe('DhcpClientClassSetFormComponent', () => {
    let component: DhcpClientClassSetFormComponent
    let fixture: ComponentFixture<DhcpClientClassSetFormComponent>
    let fb: FormBuilder = new FormBuilder()

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [HelpTipComponent, DhcpClientClassSetFormComponent],
            imports: [
                ButtonModule,
                FormsModule,
                NoopAnimationsModule,
                ReactiveFormsModule,
                TableModule,
                FloatLabelModule,
                AutoCompleteModule,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(DhcpClientClassSetFormComponent)
        component = fixture.componentInstance
        component.classFormControl = fb.control(null)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should sort client classes', () => {
        component.clientClasses = [
            {
                name: 'router',
            },
            {
                name: 'cable-modem',
            },
            {
                name: 'DROP',
            },
        ]
        fixture.detectChanges()

        expect(component.sortedClientClasses).toBeTruthy()
        expect(component.sortedClientClasses.length).toBe(3)
        expect(component.sortedClientClasses[0].name).toBe('cable-modem')
        expect(component.sortedClientClasses[1].name).toBe('DROP')
        expect(component.sortedClientClasses[2].name).toBe('router')
    })

    it('should display and insert class list', () => {
        component.clientClasses = [
            {
                name: 'router',
            },
            {
                name: 'cable-modem',
            },
            {
                name: 'DROP',
            },
        ]
        fixture.detectChanges()

        const autoCompleteDe = fixture.debugElement.query(By.directive(AutoComplete))
        expect(autoCompleteDe).toBeTruthy()
        const dropdownButton = autoCompleteDe.query(By.css('.p-autocomplete-dropdown'))
        expect(dropdownButton).toBeTruthy()
        dropdownButton.nativeElement.click()
        fixture.detectChanges()

        const classSpans = fixture.debugElement.queryAll(By.css('ul.p-autocomplete-list li span'))
        expect(classSpans).toBeTruthy()
        expect(classSpans.length).toBe(3)
        expect(classSpans.map((de) => de.nativeElement.innerText)).toEqual(
            jasmine.arrayContaining(['router', 'cable-modem', 'DROP'])
        )

        classSpans[0].parent.nativeElement.click()
        fixture.detectChanges()

        expect(component.classFormControl.value).toBeTruthy()
        expect(component.classFormControl.value.length).toEqual(1)
        expect(component.classFormControl.value[0]).toEqual(classSpans[0].nativeElement.innerText)
    })

    it('should handle empty class list', () => {
        const autoCompleteDe = fixture.debugElement.query(By.directive(AutoComplete))
        expect(autoCompleteDe).toBeTruthy()
        const dropdownButton = autoCompleteDe.query(By.css('.p-autocomplete-dropdown'))
        expect(dropdownButton).toBeTruthy()
        dropdownButton.nativeElement.click()
        fixture.detectChanges()

        const classListItems = fixture.debugElement.queryAll(By.css('ul.p-autocomplete-list li'))
        expect(classListItems).toBeTruthy()
        expect(classListItems.length).toBe(1)
        expect(classListItems[0].nativeElement.innerText).toMatch('No results found')
    })
})
