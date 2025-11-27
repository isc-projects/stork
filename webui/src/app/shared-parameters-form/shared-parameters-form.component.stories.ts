import { Meta, StoryObj } from '@storybook/angular'
import { SharedParametersFormComponent } from './shared-parameters-form.component'
import { FormControl, FormGroup } from '@angular/forms'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { StorkValidators } from '../validators'

interface SubnetForm {
    allocator: SharedParameterFormGroup<string>
    cacheMaxAge: SharedParameterFormGroup<number>
    cacheThreshold: SharedParameterFormGroup<number>
    ddnsGeneratedPrefix: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate: SharedParameterFormGroup<boolean>
    hostReservationIdentifiers: SharedParameterFormGroup<string[]>
    requireClientClasses: SharedParameterFormGroup<string[]>
    relayAddresses: SharedParameterFormGroup<string[]>
}

export default {
    title: 'App/SharedParametersForm',
    component: SharedParametersFormComponent,
    argTypes: {
        formGroup: {
            table: {
                disable: true,
            },
        },
    },
} as Meta

type Story = StoryObj<SharedParametersFormComponent<SubnetForm>>

export const VariousParameters: Story = {
    args: {
        servers: ['server 1', 'server 2'],
        formGroup: new FormGroup<SubnetForm>({
            allocator: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['iterative', 'random', 'flq'],
                },
                [new FormControl<string>('iterative'), new FormControl<string>(null)]
            ),

            cacheMaxAge: new SharedParameterFormGroup(
                {
                    type: 'number',
                },
                [new FormControl(1000), new FormControl(2000)]
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
            ddnsOverrideClientUpdate: new SharedParameterFormGroup(
                {
                    type: 'boolean',
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
            relayAddresses: new SharedParameterFormGroup(
                {
                    type: 'string',
                    isArray: true,
                },
                [
                    new FormControl<string[]>(['192.0.2.1', '192.0.2.2', '192.0.2.3'], StorkValidators.ipv4()),
                    new FormControl<string[]>(['192.0.2.1', '192.0.2.2'], StorkValidators.ipv4()),
                ]
            ),
            requireClientClasses: new SharedParameterFormGroup(
                {
                    type: 'client-classes',
                },
                [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
            ),
        }),
    },
}
