import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form.component'

import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { provideAnimations } from '@angular/platform-browser/animations'
import { FormBuilder } from '@angular/forms'

export default {
    title: 'App/DhcpClientClassSetForm',
    component: DhcpClientClassSetFormComponent,
    decorators: [
        applicationConfig({
            providers: [provideAnimations()],
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
