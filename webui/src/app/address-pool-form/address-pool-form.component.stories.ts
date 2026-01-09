import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { AddressPoolFormComponent } from './address-pool-form.component'
import { toastDecorator } from '../utils-stories'
import { FormControl, FormGroup, UntypedFormArray } from '@angular/forms'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { MessageService } from 'primeng/api'
import { AddressPoolForm, AddressRangeForm, KeaPoolParametersForm } from '../forms/subnet-set-form.service'
import { StorkValidators } from '../validators'

export default {
    title: 'App/AddressPoolForm',
    component: AddressPoolFormComponent,
    argTypes: {
        formGroup: {
            table: {
                disable: true,
            },
        },
    },
    decorators: [
        applicationConfig({
            providers: [MessageService],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<AddressPoolFormComponent>

export const AddressPool4: Story = {
    args: {
        subnet: '192.0.2.0/24',
        formGroup: new FormGroup<AddressPoolForm>({
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
                poolID: new SharedParameterFormGroup(
                    {
                        type: 'number',
                    },
                    [new FormControl(123), new FormControl(123)]
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
        }),
        selectableDaemons: [
            {
                id: 1,
                name: 'dhcp4',
                version: '2.7.0',
                label: 'first/dhcp4',
            },
            {
                id: 2,
                name: 'dhcp4',
                version: '2.7.0',
                label: 'second/dhcp4',
            },
        ],
    },
}

export const AddressPool6: Story = {
    args: {
        subnet: '2001:db8:1::/64',
        formGroup: new FormGroup<AddressPoolForm>({
            range: new FormGroup<AddressRangeForm>(
                {
                    start: new FormControl<string>('2001:db8:1::10', StorkValidators.ipInSubnet('2001:db8:1::/64')),
                    end: new FormControl<string>('2001:db8:1::ffff', StorkValidators.ipInSubnet('2001:db8:1::/64')),
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
                poolID: new SharedParameterFormGroup(
                    {
                        type: 'number',
                    },
                    [new FormControl(123), new FormControl(123)]
                ),
                requireClientClasses: new SharedParameterFormGroup(
                    {
                        type: 'client-classes',
                    },
                    [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
                ),
            }),
            options: new FormGroup({
                unlocked: new FormControl(false),
                data: new UntypedFormArray([new UntypedFormArray([]), new UntypedFormArray([])]),
            }),
            selectedDaemons: new FormControl<number[]>([1, 2]),
        }),
        selectableDaemons: [
            {
                id: 1,
                name: 'dhcp6',
                version: '2.7.0',
                label: 'first/dhcp6',
            },
            {
                id: 2,
                name: 'dhcp6',
                version: '2.7.0',
                label: 'second/dhcp6',
            },
        ],
    },
}
