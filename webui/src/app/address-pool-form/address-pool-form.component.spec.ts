import { ComponentFixture, TestBed } from '@angular/core/testing'

import { AddressPoolFormComponent } from './address-pool-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { FormControl, FormGroup, FormsModule, ReactiveFormsModule, UntypedFormArray } from '@angular/forms'
import { InputNumberModule } from 'primeng/inputnumber'
import { MultiSelectModule } from 'primeng/multiselect'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { SplitButtonModule } from 'primeng/splitbutton'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { ToastModule } from 'primeng/toast'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { AddressPoolForm, AddressRangeForm, KeaPoolParametersForm } from '../forms/subnet-set-form.service'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { DividerModule } from 'primeng/divider'
import { By } from '@angular/platform-browser'
import { StorkValidators } from '../validators'

describe('AddressPoolFormComponent', () => {
    let component: AddressPoolFormComponent
    let fixture: ComponentFixture<AddressPoolFormComponent>

    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DividerModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                InputNumberModule,
                MultiSelectModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                SplitButtonModule,
                TableModule,
                TagModule,
                ToastModule,
            ],
            declarations: [
                AddressPoolFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                HelpTipComponent,
                SharedParametersFormComponent,
            ],
        })
        fixture = TestBed.createComponent(AddressPoolFormComponent)
        component = fixture.componentInstance
        component.subnet = '192.0.2.0/24'
        component.formGroup = new FormGroup<AddressPoolForm>({
            range: new FormGroup<AddressRangeForm>(
                {
                    start: new FormControl<string>('192.0.2.10', StorkValidators.ipInSubnet('192.0.2.0/24')),
                    end: new FormControl<string>('192.0.2.100', StorkValidators.ipInSubnet('192.0.2.0/24')),
                },
                StorkValidators.ipRangeBounds
            ),
            parameters: new FormGroup<KeaPoolParametersForm>({
                clientClass: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                    },
                    [new FormControl<string>('foo'), new FormControl<string>('bar')]
                ),
                requireClientClasses: new SharedParameterFormGroup(
                    {
                        type: 'client-classes',
                    },
                    [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
                ),
            }),
            options: new FormGroup({
                unlocked: new FormControl(true),
                data: new UntypedFormArray([
                    new UntypedFormArray([
                        new FormGroup({
                            alwaysSend: new FormControl(false),
                            optionCode: new FormControl(5),
                            optionFields: new UntypedFormArray([]),
                            suboptions: new UntypedFormArray([]),
                        }),
                    ]),
                    new UntypedFormArray([
                        new FormGroup({
                            alwaysSend: new FormControl(false),
                            optionCode: new FormControl(6),
                            optionFields: new UntypedFormArray([]),
                            suboptions: new UntypedFormArray([]),
                        }),
                    ]),
                ]),
            }),
            selectedDaemons: new FormControl<number[]>([1, 2]),
        })
        component.selectableDaemons = [
            {
                id: 1,
                appId: 1,
                appType: 'kea',
                name: 'first/dhcp4',
                version: '3.0.0',
                label: 'first/dhcp4',
            },
            {
                id: 2,
                appId: 2,
                appType: 'kea',
                name: 'second/dhcp4',
                version: '3.0.0',
                label: 'second/dhcp4',
            },
            {
                id: 3,
                appId: 3,
                appType: 'kea',
                name: 'third/dhcp4',
                version: '3.0.0',
                label: 'third/dhcp4',
            },
        ]
        component.ngOnInit()
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should initialize the servers selector', () => {
        expect(component.servers.length).toBe(2)
        expect(component.servers[0]).toBe('first/dhcp4')
        expect(component.servers[1]).toBe('second/dhcp4')
    })

    it('should reduce the form for unselected server', () => {
        component.formGroup.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        fixture.detectChanges()

        expect(component.servers.length).toBe(1)
        expect(component.servers[0]).toBe('second/dhcp4')

        expect(component.formGroup.get('range.start')?.value).toBe('192.0.2.10')
        expect(component.formGroup.get('range.end')?.value).toBe('192.0.2.100')

        const clientClass = component.formGroup.get('parameters.clientClass.values') as UntypedFormArray
        expect(clientClass).toBeTruthy()
        expect(clientClass.length).toBe(1)
        expect(clientClass.get('0').value).toBe('bar')

        const requireClientClasses = component.formGroup.get(
            'parameters.requireClientClasses.values'
        ) as UntypedFormArray
        expect(requireClientClasses).toBeTruthy()
        expect(requireClientClasses.length).toBe(1)
        expect(requireClientClasses.get('0').value).toEqual(['foo', 'bar', 'auf'])

        const options = component.formGroup.get('options.data') as UntypedFormArray
        expect(options).toBeTruthy()
        expect(options.get('0.0.optionCode')?.value).toBe(6)
    })

    it('should extend the form for newly selected server', () => {
        component.formGroup.get('selectedDaemons').setValue([1, 2, 3])
        component.onDaemonsChange({
            itemValue: 3,
        })
        fixture.detectChanges()

        expect(component.servers.length).toBe(3)
        expect(component.servers[0]).toBe('first/dhcp4')
        expect(component.servers[1]).toBe('second/dhcp4')
        expect(component.servers[2]).toBe('third/dhcp4')

        expect(component.formGroup.get('range.start')?.value).toBe('192.0.2.10')
        expect(component.formGroup.get('range.end')?.value).toBe('192.0.2.100')

        const clientClass = component.formGroup.get('parameters.clientClass.values') as UntypedFormArray
        expect(clientClass).toBeTruthy()
        expect(clientClass.length).toBe(3)
        expect(clientClass.get('0').value).toBe('foo')
        expect(clientClass.get('1').value).toBe('bar')
        expect(clientClass.get('2').value).toBe('foo')

        const requireClientClasses = component.formGroup.get(
            'parameters.requireClientClasses.values'
        ) as UntypedFormArray
        expect(requireClientClasses).toBeTruthy()
        expect(requireClientClasses.length).toBe(3)
        expect(requireClientClasses.get('0').value).toEqual(['foo', 'bar'])
        expect(requireClientClasses.get('1').value).toEqual(['foo', 'bar', 'auf'])
        expect(requireClientClasses.get('2').value).toEqual(['foo', 'bar'])

        const options = component.formGroup.get('options.data') as UntypedFormArray
        expect(options).toBeTruthy()
        expect(options.get('0.0.optionCode')?.value).toBe(5)
        expect(options.get('1.0.optionCode')?.value).toBe(6)
        expect(options.get('2.0.optionCode')?.value).toBe(5)
    })

    it('should reset a form when all servers are unselected', () => {
        component.formGroup.get('selectedDaemons').setValue([])
        component.onDaemonsChange({
            itemValue: 2,
        })
        fixture.detectChanges()

        expect(component.servers.length).toBe(0)

        expect(component.formGroup.get('range.start')?.value).toBe('192.0.2.10')
        expect(component.formGroup.get('range.end')?.value).toBe('192.0.2.100')

        const clientClass = component.formGroup.get('parameters.clientClass.values') as UntypedFormArray
        expect(clientClass).toBeTruthy()
        expect(clientClass.length).toBe(1)
        expect(clientClass.get('0').value).toBeFalsy()

        const requireClientClasses = component.formGroup.get(
            'parameters.requireClientClasses.values'
        ) as UntypedFormArray
        expect(requireClientClasses).toBeTruthy()
        expect(requireClientClasses.length).toBe(1)
        expect(requireClientClasses.get('0').value).toEqual([])

        const options = component.formGroup.get('options.data') as UntypedFormArray
        expect(options).toBeTruthy()
        expect(options.length).toBe(1)
        expect((options.get('0') as UntypedFormArray)?.length).toBe(0)
    })

    it('should validate the address range', () => {
        component.formGroup.get('range.start').setValue('192.0.2.10')
        component.formGroup.get('range.end').setValue('192.0.2.5')
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)

        expect(smalls[0].nativeElement.innerText).toBe(
            'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.'
        )
    })

    it('should validate the lower bound address', () => {
        component.formGroup.get('range.start').setValue('192.0.1.10')
        component.formGroup.get('range.start').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain('192.0.1.10 does not belong to subnet 192.0.2.0/24.')
    })

    it('should validate the upper bound address', () => {
        component.formGroup.get('range.end').setValue('192.0.8.10')
        component.formGroup.get('range.end').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain('192.0.8.10 does not belong to subnet 192.0.2.0/24.')
    })

    it('should contain server assignment multi select', () => {
        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const assignments = panels[1].query(By.css('p-multiSelect'))
        expect(assignments).toBeTruthy()
    })

    it('should contain pool specific parameters', () => {
        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const params = panels[2].query(By.css('app-shared-parameters-form'))
        expect(params).toBeTruthy()
        expect(params.nativeElement.innerText).toContain('Client Class')
        expect(params.nativeElement.innerText).toContain('Require Client Classes')
    })

    it('should contain DHCP options', () => {
        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const options = panels[3].query(By.css('app-dhcp-option-set-form'))
        expect(options).toBeTruthy()
        expect(options.nativeElement.innerText).toContain('Empty Option')
    })
})
