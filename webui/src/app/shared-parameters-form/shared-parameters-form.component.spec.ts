import { ComponentFixture, TestBed } from '@angular/core/testing'

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
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MultiSelectModule } from 'primeng/multiselect'

/**
 * Intrface to the form used in the unit tests.
 */
interface SubnetForm {
    allocator?: SharedParameterFormGroup<string>
    cacheMaxAge?: SharedParameterFormGroup<number>
    cacheThreshold?: SharedParameterFormGroup<number>
    ddnsGeneratedPrefix?: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate?: SharedParameterFormGroup<boolean>
    dhcpDdnsEnableUpdates?: SharedParameterFormGroup<boolean>
    hostReservationIdentifiers?: SharedParameterFormGroup<string[]>
    requireClientClasses?: SharedParameterFormGroup<string[]>
    relayAddresses?: SharedParameterFormGroup<string[]>
}

describe('SharedParametersFormComponent', () => {
    let component: SharedParametersFormComponent<SubnetForm>
    let fixture: ComponentFixture<SharedParametersFormComponent<SubnetForm>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                ArrayValueSetFormComponent,
                DhcpClientClassSetFormComponent,
                HelpTipComponent,
                SharedParametersFormComponent,
            ],
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FormsModule,
                InputNumberModule,
                MultiSelectModule,
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
        component.servers = ['server 1', 'server 2']
        component.formGroup = new FormGroup<SubnetForm>({
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
            dhcpDdnsEnableUpdates: new SharedParameterFormGroup(
                {
                    type: 'boolean',
                    required: true,
                },
                [new FormControl<boolean>(true), new FormControl<boolean>(true)]
            ),
            hostReservationIdentifiers: new SharedParameterFormGroup(
                {
                    type: 'string',
                    isArray: true,
                    values: ['hw-address', 'client-id', 'duid', 'circuit-id'],
                },
                [new FormControl(['hw-address']), new FormControl(['hw-address'])]
            ),
            requireClientClasses: new SharedParameterFormGroup(
                {
                    type: 'client-classes',
                },
                [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
            ),
            relayAddresses: new SharedParameterFormGroup(
                {
                    type: 'string',
                    isArray: true,
                },
                [
                    new FormControl(['192.0.2.1', '192.0.2.2'], StorkValidators.ipv4),
                    new FormControl(['192.0.2.1', '192.0.2.2', '192.0.2.3']),
                ]
            ),
        })
        fixture.detectChanges()

        // Make sure the keys are sorted.
        expect(component.parameterNames).toEqual([
            'allocator',
            'cacheMaxAge',
            'cacheThreshold',
            'ddnsGeneratedPrefix',
            'ddnsOverrideClientUpdate',
            'dhcpDdnsEnableUpdates',
            'hostReservationIdentifiers',
            'relayAddresses',
            'requireClientClasses',
        ])

        // Validate the section header.
        let divs = fixture.debugElement.queryAll(By.css('.shared-parameter-wrapper.font-semibold > div'))
        // First two divs are hidden for wider viewports, but visible for smaller viewports.
        expect(divs.length).toBe(5)
        // Check last three divs visible for larger viewports.
        expect(divs[2].nativeElement.innerText).toBe('Parameter')
        expect(divs[3].nativeElement.innerText).toBe('Value')
        expect(divs[4].nativeElement.innerText).toBe('Unlock')

        let allWrapperDivs = fixture.debugElement.queryAll(By.css('.shared-parameter-wrapper:not(.font-semibold)'))
        expect(allWrapperDivs.length).toBe(9)

        // Allocator.
        let labelDiv = allWrapperDivs[0].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Allocator')
        let controls = allWrapperDivs[0].queryAll(By.css('p-dropdown'))
        expect(controls.length).toBe(2)
        let tags = allWrapperDivs[0].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        let btns = allWrapperDivs[0].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        let checkbox = allWrapperDivs[0].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Cache Max Age.
        labelDiv = allWrapperDivs[1].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Cache Max Age')
        controls = allWrapperDivs[1].queryAll(By.css('p-inputNumber'))
        expect(controls.length).toBe(2)
        tags = allWrapperDivs[1].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = allWrapperDivs[1].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[1].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Cache Threshold.
        labelDiv = allWrapperDivs[2].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Cache Threshold')
        controls = allWrapperDivs[2].queryAll(By.css('p-inputNumber'))
        expect(controls.length).toBe(2)
        tags = allWrapperDivs[2].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = allWrapperDivs[2].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[2].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // DDNS Generated Prefix.
        labelDiv = allWrapperDivs[3].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('DDNS Generated Prefix')
        controls = allWrapperDivs[3].queryAll(By.css('input.p-inputtext'))
        expect(controls.length).toBe(2)
        tags = allWrapperDivs[3].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = allWrapperDivs[3].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[3].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // DDNS Override Client Update.
        labelDiv = allWrapperDivs[4].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('DDNS Override Client Update')
        controls = allWrapperDivs[4].queryAll(By.css('p-triStateCheckbox'))
        expect(controls.length).toBe(1)
        tags = allWrapperDivs[4].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(0)
        btns = allWrapperDivs[4].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(1)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[4].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // DHCP DDNS Enable Updates.
        labelDiv = allWrapperDivs[5].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('DHCP DDNS Enable Updates')
        controls = allWrapperDivs[5].queryAll(By.css('p-checkbox'))
        expect(controls.length).toBe(3)
        tags = allWrapperDivs[5].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(0)
        btns = allWrapperDivs[5].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(1)
        expect(btns[0].nativeElement.innerText).toBe('Clear')

        // Host Reservation Identifiers.
        labelDiv = allWrapperDivs[6].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Host Reservation Identifiers')
        controls = allWrapperDivs[6].queryAll(By.css('p-multiSelect'))
        expect(controls.length).toBe(1)
        tags = allWrapperDivs[6].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(0)
        btns = allWrapperDivs[6].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(1)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[6].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Relay
        labelDiv = allWrapperDivs[7].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Relay Addresses')
        controls = allWrapperDivs[7].queryAll(By.css('app-array-value-set-form'))
        expect(controls.length).toBe(2)
        tags = allWrapperDivs[7].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = allWrapperDivs[7].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[7].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()

        // Require Client Classes.
        labelDiv = allWrapperDivs[8].queryAll(By.css('div.font-semibold'))
        expect(labelDiv.length).toBe(1)
        expect(labelDiv[0].nativeElement.innerText).toBe('Require Client Classes')
        controls = allWrapperDivs[8].queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(controls.length).toBe(2)
        tags = allWrapperDivs[8].queryAll(By.css('p-tag'))
        expect(tags.length).toBe(2)
        expect(tags[0].nativeElement.innerText).toBe('server 1')
        expect(tags[1].nativeElement.innerText).toBe('server 2')
        btns = allWrapperDivs[8].queryAll(By.css('[label=Clear]'))
        expect(btns.length).toBe(2)
        expect(btns[0].nativeElement.innerText).toBe('Clear')
        expect(btns[1].nativeElement.innerText).toBe('Clear')
        checkbox = allWrapperDivs[8].query(By.css('p-checkbox'))
        expect(checkbox).toBeTruthy()
    })

    it('should display use a list of selectable client classes', () => {
        component.servers = ['server 1']
        component.formGroup = new FormGroup<SubnetForm>({
            requireClientClasses: new SharedParameterFormGroup(
                {
                    type: 'client-classes',
                },
                [new FormControl([])]
            ),
        })
        component.clientClasses = [
            {
                name: 'foo',
            },
            {
                name: 'bar',
            },
        ]
        fixture.detectChanges()

        // Validate the section header.
        let divs = fixture.debugElement.queryAll(By.css('.shared-parameter-wrapper.font-semibold > div'))
        // First div is hidden for wider viewports, but visible for smaller viewports.
        expect(divs.length).toBe(3)
        // Check last two divs visible for larger viewports.
        expect(divs[1].nativeElement.innerText).toBe('Parameter')
        expect(divs[2].nativeElement.innerText).toBe('Value')

        // Require Client Classes.
        divs = fixture.debugElement.queryAll(By.css('.shared-parameter-wrapper:not(.font-semibold) > div'))
        expect(divs.length).toBe(2)
        expect(divs[0].childNodes[0].nativeNode.innerText).toBe('Require Client Classes')
        const controls = divs[1].queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(controls.length).toBe(1)

        // Click the List button to list the classes.
        const btns = divs[1].queryAll(By.css('[label=List]'))
        expect(btns.length).toBe(1)
        expect(btns[0].nativeElement.innerText).toBe('List')
        btns[0].nativeElement.click()
        fixture.detectChanges()

        // The expanded list should contain the client classes.
        expect(controls[0].nativeElement.innerText).toContain('foo')
        expect(controls[0].nativeElement.innerText).toContain('bar')
    })

    it('should clear selected value', () => {
        component.servers = ['server 1', 'server 2']
        component.formGroup = new FormGroup<SubnetForm>({
            ddnsGeneratedPrefix: new SharedParameterFormGroup(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid prefix.',
                },
                [new FormControl('myhost', StorkValidators.fqdn), new FormControl('hishost', StorkValidators.fqdn)]
            ),
        })
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
        component.servers = ['server 1', 'server 2']
        component.formGroup = new FormGroup<SubnetForm>({
            ddnsGeneratedPrefix: new SharedParameterFormGroup(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid prefix.',
                },
                [new FormControl('myhost', StorkValidators.fqdn), new FormControl('myhost', StorkValidators.fqdn)]
            ),
        })
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
        component.servers = ['server 1']
        component.formGroup = new FormGroup<SubnetForm>({
            ddnsGeneratedPrefix: new SharedParameterFormGroup(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid prefix.',
                },
                [new FormControl('myhost', StorkValidators.fqdn)]
            ),
        })
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
        component.servers = ['server 1']
        component.formGroup = new FormGroup<SubnetForm>({})
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('No parameters configured.')
    })
})
