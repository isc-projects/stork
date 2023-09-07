import { moduleMetadata, Meta, Story } from '@storybook/angular'
import { SharedParametersFormComponent } from './shared-parameters-form.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TableModule } from 'primeng/table'
import { FormControl, FormGroup, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { InputNumberModule } from 'primeng/inputnumber'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { ChipsModule } from 'primeng/chips'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ButtonModule } from 'primeng/button'
import { DropdownModule } from 'primeng/dropdown'
import { TagModule } from 'primeng/tag'
import { SharedParameterFormGroup } from '../forms/shared-parameter-form-group'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { CheckboxModule } from 'primeng/checkbox'
import { StorkValidators } from '../validators'

interface SubnetForm {
    allocator: SharedParameterFormGroup<string>
    cacheMaxAge: SharedParameterFormGroup<number>
    cacheThreshold: SharedParameterFormGroup<number>
    ddnsGeneratedPrefix: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate: SharedParameterFormGroup<boolean>
    requireClientClasses: SharedParameterFormGroup<string[]>
}

export default {
    title: 'App/SharedParametersForm',
    component: SharedParametersFormComponent,
    decorators: [
        moduleMetadata({
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
            declarations: [SharedParametersFormComponent, DhcpClientClassSetFormComponent],
            providers: [],
        }),
    ],
    argTypes: {
        formGroup: {
            table: {
                disable: true,
            },
        },
    },
} as Meta

const Template: Story<SharedParametersFormComponent<SubnetForm>> = (
    args: SharedParametersFormComponent<SubnetForm>
) => ({
    props: args,
})

export const VariousParameters = Template.bind({})
VariousParameters.args = {
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
        requireClientClasses: new SharedParameterFormGroup(
            {
                type: 'client-classes',
            },
            [new FormControl(['foo', 'bar']), new FormControl(['foo', 'bar', 'auf'])]
        ),
    }),
}
