import { ComponentFixture, TestBed } from '@angular/core/testing'
import { UntypedFormBuilder } from '@angular/forms'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { IPType } from '../iptype'

describe('DhcpOptionSetFormComponent', () => {
    let component: DhcpOptionSetFormComponent
    let fixture: ComponentFixture<DhcpOptionSetFormComponent>
    let fb: UntypedFormBuilder

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [UntypedFormBuilder, provideNoopAnimations()],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(DhcpOptionSetFormComponent)
        component = fixture.componentInstance
        fb = new UntypedFormBuilder()
        component.formArray = fb.array([])
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should add an option', () => {
        let addBtn = fixture.debugElement.query(By.css('[label="Add Option"]'))
        expect(addBtn).toBeTruthy()

        spyOn(component.optionAdd, 'emit').and.callFake(() => {
            component.formArray.push(createDefaultDhcpOptionFormGroup(IPType.IPv4))
        })

        addBtn.nativeElement.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.optionAdd.emit).toHaveBeenCalled()

        expect(component.formArray.length).toBe(1)
        expect(fixture.debugElement.query(By.css('app-dhcp-option-form'))).toBeTruthy()

        addBtn = fixture.debugElement.query(By.css('[label="Add More Options"]'))
        expect(addBtn).toBeTruthy()

        addBtn.nativeElement.dispatchEvent(new Event('click'))
        fixture.detectChanges()
        expect(component.formArray.length).toBe(2)

        component.onOptionDelete(0)
        fixture.detectChanges()
        expect(component.formArray.length).toBe(1)

        component.onOptionDelete(0)
        fixture.detectChanges()
        expect(component.formArray.length).toBe(0)
    })

    it('should lack the button for higher nesting levels', () => {
        component.nestLevel = 1
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('button'))).toBeFalsy()
    })
})
