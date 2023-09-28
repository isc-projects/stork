import { ComponentFixture, TestBed, tick } from '@angular/core/testing'

import { SharedParametersFormComponent } from './shared-parameters-form.component'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DropdownModule } from 'primeng/dropdown'
import {
    FormControl,
    FormGroup,
    FormsModule,
    ReactiveFormsModule,
    UntypedFormArray,
    UntypedFormControl,
} from '@angular/forms'
import { InputNumberModule } from 'primeng/inputnumber'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { StorkValidators } from '../validators'
import { By } from '@angular/platform-browser'

/**
 * Intrface to the form used in the unit tests.
 */
interface SubnetForm {
    allocator?: SharedParameterFormGroup<string>
    cacheMaxAge?: SharedParameterFormGroup<number>
    cacheThreshold?: SharedParameterFormGroup<number>
    ddnsGeneratedPrefix?: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate?: SharedParameterFormGroup<boolean>
    requireClientClasses?: SharedParameterFormGroup<string[]>
}

describe('SharedParametersFormComponent', () => {
    let component: SharedParametersFormComponent<SubnetForm>
    let fixture: ComponentFixture<SharedParametersFormComponent<SubnetForm>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [DhcpClientClassSetFormComponent, SharedParametersFormComponent],
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FormsModule,
                InputNumberModule,
                NoopAnimationsModule,
                TableModule,
                TagModule,
                TriStateCheckboxModule,
                OverlayPanelModule,
                ReactiveFormsModule,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedParametersFormComponent<SubnetForm>)
        component = fixture.componentInstance
        component.formGroup = new FormGroup<SubnetForm>({})
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display the form with the parameters', () => {
        ;(component.servers = ['server 1', 'server 2']),
            (component.formGroup = new FormGroup<SubnetForm>({
                ddnsOverrideClientUpdate: new SharedParameterFormGroup(
                    {
                        type: 'boolean',
                    },
                    [new FormControl<boolean>(true), new FormControl<boolean>(true)]
                ),
                cacheMaxAge: new SharedParameterFormGroup(
                    {
                        type: 'number',
                    },
                    [new FormControl(1000), new FormControl(2000)]
                ),
                allocator: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                    },
                    [new FormControl<string>('iterative'), new FormControl<string>(null)]
                ),
                cacheThreshold: new SharedParameterFormGroup(
                    {
                        type: 'number',
                        min: 0,
                        max: 1,
                        fractionDigits: 2,
                    },
                    [new FormControl(0.25), new FormControl(0.5)]
                ),
                ddnsGeneratedPrefix: new SharedParameterFormGroup(
                    {
                        type: 'string',
                        invalidText: 'Please specify a valid prefix.',
                    },
                    [new FormControl('myhost', StorkValidators.fqdn), new FormControl('hishost', StorkValidators.fqdn)]
                ),
                requireClientClasses: new SharedParameterFormGroup(
                    {
                        type: 'client-classes',
                    },
                    [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
                ),
            }))
        fixture.detectChanges()

        // Make sure the keys are sorted.
        expect(component.parameterNames).toEqual([
            'allocator',
            'cacheMaxAge',
            'cacheThreshold',
            'ddnsGeneratedPrefix',
            'ddnsOverrideClientUpdate',
            'requireClientClasses',
        ])

        let allRows = fixture.debugElement.queryAll(By.css('tr'))
        expect(allRows.length).toBe(7)

        // Validate the table header.
        let cells = allRows[0].queryAll(By.css('th'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('Parameter')
        expect(cells[1].nativeElement.innerText).toBe('Value')
        expect(cells[2].nativeElement.innerText).toBe('Unlock')

        // Allocator.
        cells = allRows[1].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('Allocator')
        let controls = cells[1].queryAll(By.css('p-dropdown'))
        expect(controls.length).toBe(2)
        let tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        let btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        let checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Cache Max Age.
        cells = allRows[2].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('Cache Max Age')
        controls = cells[1].queryAll(By.css('p-inputNumber'))
        expect(controls.length).toBe(2)
        tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Cache Threshold.
        cells = allRows[3].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('Cache Threshold')
        controls = cells[1].queryAll(By.css('p-inputNumber'))
        expect(controls.length).toBe(2)
        tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // DDNS Generated Prefix.
        cells = allRows[4].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('DDNS Generated Prefix')
        controls = cells[1].queryAll(By.css('input'))
        expect(controls.length).toBe(2)
        tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // DDNS Override Client Update.
        cells = allRows[5].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('DDNS Override Client Update')
        controls = cells[1].queryAll(By.css('p-triStateCheckbox'))
        expect(controls.length).toBe(1)
        tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(0)
        btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(1)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Require Client Classes.
        cells = allRows[6].queryAll(By.css('td'))
        expect(cells.length).toBe(3)
        expect(cells[0].nativeElement.innerText).toBe('Require Client Classes')
        controls = cells[1].queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(controls.length).toBe(2)
        tags = cells[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = cells[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = cells[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()
    })

    it('should clear selected value', () => {
        ;(component.servers = ['server 1', 'server 2']),
            (component.formGroup = new FormGroup<SubnetForm>({
                ddnsGeneratedPrefix: new SharedParameterFormGroup(
                    {
                        type: 'string',
                        invalidText: 'Please specify a valid prefix.',
                    },
                    [new FormControl('myhost', StorkValidators.fqdn), new FormControl('hishost', StorkValidators.fqdn)]
                ),
            }))
        fixture.detectChanges()

        let clearBtns = fixture.debugElement.queryAll(By.css('[label=Clear]'))
        expect(clearBtns.length).toBe(2)

        clearBtns[0].nativeElement.click()
        fixture.detectChanges()

        let parameterControls = component.getParameterFormControls('ddnsGeneratedPrefix')
        let valuesControls = (parameterControls?.get('values') as UntypedFormArray)?.controls
        expect(valuesControls?.length).toBe(2)
        expect(valuesControls[0].value).toBeFalsy()
        expect(valuesControls[1].value).toBe('hishost')
    })

    it('should unlock parameter for edit for different servers', () => {
        ;(component.servers = ['server 1', 'server 2']),
            (component.formGroup = new FormGroup<SubnetForm>({
                ddnsGeneratedPrefix: new SharedParameterFormGroup(
                    {
                        type: 'string',
                        invalidText: 'Please specify a valid prefix.',
                    },
                    [new FormControl('myhost', StorkValidators.fqdn), new FormControl('myhost', StorkValidators.fqdn)]
                ),
            }))
        fixture.detectChanges()

        let tags = fixture.debugElement.queryAll(By.css('p-tag'))
        expect(tags.length).toBe(0)

        let parameterControls = component.getParameterFormControls('ddnsGeneratedPrefix')
        let unlockControl = parameterControls?.get('unlocked') as UntypedFormControl
        unlockControl.setValue(true)
        fixture.detectChanges()

        tags = fixture.debugElement.queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
    })

    it('should validate a string value', () => {
        ;(component.servers = ['server 1']),
            (component.formGroup = new FormGroup<SubnetForm>({
                ddnsGeneratedPrefix: new SharedParameterFormGroup(
                    {
                        type: 'string',
                        invalidText: 'Please specify a valid prefix.',
                    },
                    [new FormControl('myhost', StorkValidators.fqdn)]
                ),
            }))
        fixture.detectChanges()

        let parameterControls = component.getParameterFormControls('ddnsGeneratedPrefix')
        let valuesControls = (parameterControls?.get('values') as UntypedFormArray)?.controls
        expect(valuesControls).toBeTruthy()

        valuesControls[0].setValue('-invalid.prefix')
        valuesControls[0].markAsTouched()
        valuesControls[0].markAsDirty()
        fixture.detectChanges()

        expect(valuesControls[0].valid).toBeFalse()

        let errorHint = fixture.debugElement.query(By.css('small'))
        expect(errorHint).toBeTruthy()
        expect(errorHint.nativeElement.innerText).toBe('Please specify a valid prefix.')
    })

    it('should display that there are no parameters', () => {
        ;(component.servers = ['server 1']), (component.formGroup = new FormGroup<SubnetForm>({}))
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('No parameters configured.')
    })
})
