import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { CheckboxModule } from 'primeng/checkbox'
import { ButtonModule } from 'primeng/button'
import { PopoverModule } from 'primeng/popover'
import { TableModule } from 'primeng/table'
import { FloatLabelModule } from 'primeng/floatlabel'
import { AutoCompleteModule } from 'primeng/autocomplete'

export default {
    title: 'App/DhcpClientClassSetForm',
    component: DhcpClientClassSetFormComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                CheckboxModule,
                FormsModule,
                NoopAnimationsModule,
                PopoverModule,
                ReactiveFormsModule,
                TableModule,
                FloatLabelModule,
                AutoCompleteModule,
            ],
            declarations: [DhcpClientClassSetFormComponent, HelpTipComponent],
        }),
    ],
} as Meta

const fb: FormBuilder = new FormBuilder()

type Story = StoryObj<DhcpClientClassSetFormComponent>

export const ManyClasses: Story = {
    args: {
        classFormControl: fb.control(null),
        clientClasses: [
            {
                name: 'router',
            },
            {
                name: 'cable-modem',
            },
            {
                name: 'DROP',
            },
            {
                name: 'fascinating',
            },
            {
                name: 'zeus',
            },
            {
                name: 'bad',
            },
            {
                name: 'good',
            },
            {
                name: 'unregistered',
            },
            {
                name: 'finance',
            },
            {
                name: 'corrupted',
            },
            {
                name: 'hardware',
            },
            {
                name: 'software',
            },
            {
                name: 'server',
            },
            {
                name: 'client',
            },
        ],
    },
}

export const NullClasses: Story = {
    args: {
        classFormControl: fb.control(null),
        clientClasses: null,
    },
}

export const EmptyClasses: Story = {
    args: {
        classFormControl: fb.control(null),
        clientClasses: [],
    },
}
