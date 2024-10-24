import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PrefixPoolFormComponent } from './prefix-pool-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { FormControl, FormGroup, FormsModule, ReactiveFormsModule, UntypedFormArray, Validators } from '@angular/forms'
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
import { KeaPoolParametersForm, PrefixForm, PrefixPoolForm } from '../forms/subnet-set-form.service'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { By } from '@angular/platform-browser'
import { StorkValidators } from '../validators'

describe('PrefixPoolFormComponent', () => {
    let component: PrefixPoolFormComponent
    let fixture: ComponentFixture<PrefixPoolFormComponent>

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
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                HelpTipComponent,
                PrefixPoolFormComponent,
                SharedParametersFormComponent,
            ],
        })
        fixture = TestBed.createComponent(PrefixPoolFormComponent)
        component = fixture.componentInstance
        component.subnet = '2001:db8:1::/64'
        component.formGroup = new FormGroup<PrefixPoolForm>({
            prefixes: new FormGroup<PrefixForm>(
                {
                    prefix: new FormControl(
                        '2001:db8:1::/64',
                        Validators.compose([Validators.required, StorkValidators.ipv6Prefix])
                    ),
                    delegatedLength: new FormControl(80, Validators.required),
                    excludedPrefix: new FormControl('', StorkValidators.ipv6Prefix),
                },
                Validators.compose([
                    StorkValidators.ipv6PrefixDelegatedLength,
                    StorkValidators.ipv6ExcludedPrefixDelegatedLength,
                    StorkValidators.ipv6ExcludedPrefix,
                ])
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
                name: 'first/dhcp6',
                version: '3.0.0',
                label: 'first/dhcp6',
            },
            {
                id: 2,
                appId: 2,
                appType: 'kea',
                name: 'second/dhcp6',
                version: '3.0.0',
                label: 'second/dhcp6',
            },
            {
                id: 3,
                appId: 3,
                appType: 'kea',
                name: 'third/dhcp6',
                version: '3.0.0',
                label: 'third/dhcp6',
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
        expect(component.servers[0]).toBe('first/dhcp6')
        expect(component.servers[1]).toBe('second/dhcp6')
    })

    it('should reduce the form for unselected server', () => {
        component.formGroup.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        fixture.detectChanges()

        expect(component.servers.length).toBe(1)
        expect(component.servers[0]).toBe('second/dhcp6')

        expect(component.formGroup.get('prefixes.prefix')?.value).toBe('2001:db8:1::/64')
        expect(component.formGroup.get('prefixes.delegatedLength')?.value).toBe(80)
        expect(component.formGroup.get('prefixes.excludedPrefix')?.value).toBe('')

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
        expect(component.servers[0]).toBe('first/dhcp6')
        expect(component.servers[1]).toBe('second/dhcp6')
        expect(component.servers[2]).toBe('third/dhcp6')

        expect(component.formGroup.get('prefixes.prefix')?.value).toBe('2001:db8:1::/64')
        expect(component.formGroup.get('prefixes.delegatedLength')?.value).toBe(80)
        expect(component.formGroup.get('prefixes.excludedPrefix')?.value).toBe('')

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

        expect(component.formGroup.get('prefixes.prefix')?.value).toBe('2001:db8:1::/64')
        expect(component.formGroup.get('prefixes.delegatedLength')?.value).toBe(80)
        expect(component.formGroup.get('prefixes.excludedPrefix')?.value).toBe('')

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

    it('should validate the prefix pool prefix', () => {
        component.formGroup.get('prefixes.prefix').setValue('2001:db8:1::/abc')
        component.formGroup.get('prefixes.prefix').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain('2001:db8:1::/abc is not a valid IPv6 prefix.')
    })

    it('should validate the prefix pool excluded prefix', () => {
        component.formGroup.get('prefixes.prefix').setValue('2001:db8:cafe::/56')
        component.formGroup.get('prefixes.prefix').markAsDirty()
        component.formGroup.get('prefixes.delegatedLength').setValue(60)
        component.formGroup.get('prefixes.delegatedLength').markAsDirty()
        component.formGroup.get('prefixes.excludedPrefix').setValue('2001:db8:dead::/64')
        component.formGroup.get('prefixes.excludedPrefix').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain(
            '2001:db8:dead::/64 excluded prefix is not within the 2001:db8:cafe::/56 prefix.'
        )
    })

    it('should validate the delegated length against the pool prefix', () => {
        component.formGroup.get('prefixes.prefix').setValue('2001:db8:cafe::/56')
        component.formGroup.get('prefixes.prefix').markAsDirty()
        component.formGroup.get('prefixes.delegatedLength').setValue(48)
        component.formGroup.get('prefixes.delegatedLength').markAsDirty()
        component.formGroup.get('prefixes.excludedPrefix').setValue('2001:db8:cafe::/64')
        component.formGroup.get('prefixes.excludedPrefix').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain(
            'Delegated prefix length must be greater or equal the 2001:db8:cafe::/56 prefix length.'
        )
    })

    it('should validate the delegated length against the excluded pool prefix', () => {
        component.formGroup.get('prefixes.prefix').setValue('2001:db8:cafe::/56')
        component.formGroup.get('prefixes.prefix').markAsDirty()
        component.formGroup.get('prefixes.delegatedLength').setValue(80)
        component.formGroup.get('prefixes.delegatedLength').markAsDirty()
        component.formGroup.get('prefixes.excludedPrefix').setValue('2001:db8:cafe::/64')
        component.formGroup.get('prefixes.excludedPrefix').markAsDirty()
        fixture.detectChanges()

        const panels = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(panels.length).toBe(4)

        const smalls = panels[0].queryAll(By.css('small'))
        expect(smalls.length).toBe(1)
        expect(smalls[0].nativeElement.innerText).toContain(
            'Delegated prefix length must be lower than the 2001:db8:cafe::/64 excluded prefix length.'
        )
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
