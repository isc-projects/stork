import { ComponentFixture, TestBed } from '@angular/core/testing'
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { ButtonModule } from 'primeng/button'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { By } from '@angular/platform-browser'

describe('DhcpClientClassSetFormComponent', () => {
    let component: DhcpClientClassSetFormComponent
    let fixture: ComponentFixture<DhcpClientClassSetFormComponent>
    let fb: FormBuilder = new FormBuilder()

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [HelpTipComponent, DhcpClientClassSetFormComponent],
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                FormsModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                TableModule,
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

        const listButton = fixture.debugElement.query(By.css('[label=List]'))
        expect(listButton).toBeTruthy()
        listButton.nativeElement.click()
        fixture.detectChanges()

        const checkboxes = fixture.debugElement.queryAll(By.css('p-checkbox'))
        expect(checkboxes.length).toBe(3)

        spyOn(component, 'mergeSelected')

        const insertButton = fixture.debugElement.query(By.css('[label=Insert]'))
        expect(insertButton).toBeTruthy()
        insertButton.nativeElement.click()
        fixture.detectChanges()

        expect(component.mergeSelected).toHaveBeenCalled()
    })

    it('should cancel inserting class list', () => {
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

        const listButton = fixture.debugElement.query(By.css('[label=List]'))
        expect(listButton).toBeTruthy()
        listButton.nativeElement.click()
        fixture.detectChanges()

        spyOn(component, 'mergeSelected')
        spyOn(component, 'cancelSelected')

        const cancelButton = fixture.debugElement.query(By.css('[label=Cancel]'))
        expect(cancelButton).toBeTruthy()
        cancelButton.nativeElement.click()
        fixture.detectChanges()

        expect(component.mergeSelected).not.toHaveBeenCalled()
        expect(component.cancelSelected).toHaveBeenCalled()
    })

    it('should handle empty class list', () => {
        const listButton = fixture.debugElement.query(By.css('[label=List]'))
        expect(listButton).toBeTruthy()
        listButton.nativeElement.click()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-table'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label=Insert]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label=Cancel]'))).toBeTruthy()

        const classPanel = fixture.debugElement.query(By.css('p-overlayPanel'))
        expect(classPanel).toBeTruthy()
        expect(classPanel.nativeElement.innerText).toContain('No classes found.')
    })

    it('should check that the class is in the input box', () => {
        const clientClasses = ['router', 'cable-modem', 'DROP']
        component.classFormControl.patchValue(clientClasses)
        expect(component.isUsed('router')).toBeTrue()
        expect(component.isUsed('cable-modem')).toBeTrue()
        expect(component.isUsed('DROP')).toBeTrue()
        expect(component.isUsed('other')).toBeFalse()
    })

    it('should merge selected client classes', () => {
        spyOn(component.classSelectionPanel, 'hide')

        const clientClasses = ['router', 'cable-modem', 'DROP']
        component.classFormControl.patchValue(clientClasses)
        component.selectedClientClasses = ['server', 'client', 'cable-modem']
        component.mergeSelected()

        const newValue = component.classFormControl.value as Array<string>
        expect(newValue).toBeTruthy()
        expect(newValue.length).toBe(5)
        expect(newValue[0]).toBe('router')
        expect(newValue[1]).toBe('cable-modem')
        expect(newValue[2]).toBe('DROP')
        expect(newValue[3]).toBe('server')
        expect(newValue[4]).toBe('client')

        expect(component.selectedClientClasses.length).toBe(0)
        expect(component.classSelectionPanel.hide).toHaveBeenCalled()
    })

    it('should cancel class selection and hide panel', () => {
        spyOn(component.classSelectionPanel, 'hide')
        component.selectedClientClasses = ['server', 'client', 'cable-modem']
        component.cancelSelected()

        expect(component.selectedClientClasses.length).toBe(0)
        expect(component.classSelectionPanel.hide).toHaveBeenCalled()
    })
})
