import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { PrefixPoolFormComponent } from './prefix-pool-form.component'
import { toastDecorator } from '../utils-stories'
import { FormControl, FormGroup, UntypedFormArray, Validators } from '@angular/forms'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { MessageService } from 'primeng/api'
import { provideAnimations } from '@angular/platform-browser/animations'
import { KeaPoolParametersForm, PrefixForm, PrefixPoolForm } from '../forms/subnet-set-form.service'
import { StorkValidators } from '../validators'

export default {
    title: 'App/PrefixPoolForm',
    component: PrefixPoolFormComponent,
    argTypes: {
        formGroup: {
            table: {
                disable: true,
            },
        },
    },
    decorators: [
        applicationConfig({
            providers: [provideAnimations(), MessageService],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<PrefixPoolFormComponent>

export const PrefixPool: Story = {
    args: {
        subnet: '2001:db8:1::/64',
        formGroup: new FormGroup<PrefixPoolForm>({
            prefixes: new FormGroup<PrefixForm>(
                {
                    prefix: new FormControl(
                        '2001:db8:1::/64',
                        Validators.compose([Validators.required, StorkValidators.ipv6Prefix])
                    ),
                    delegatedLength: new FormControl(96, Validators.required),
                    excludedPrefix: new FormControl('2001:db8:1:0:1:1::/98', StorkValidators.ipv6Prefix),
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
                unlocked: new FormControl(false),
                data: new UntypedFormArray([new UntypedFormArray([]), new UntypedFormArray([])]),
            }),
            selectedDaemons: new FormControl<number[]>([1, 2]),
        }),
        selectableDaemons: [
            {
                id: 1,
                appId: 1,
                appType: 'kea',
                name: 'first/dhcp6',
                version: '2.7.0',
                label: 'first/dhcp6',
            },
            {
                id: 2,
                appId: 2,
                appType: 'kea',
                name: 'second/dhcp6',
                version: '2.7.0',
                label: 'second/dhcp6',
            },
        ],
    },
}
