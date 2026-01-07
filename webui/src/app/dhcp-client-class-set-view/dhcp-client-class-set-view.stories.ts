import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view.component'

import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { toastDecorator } from '../utils-stories'
import { MessageService } from 'primeng/api'

export default {
    title: 'App/DhcpClientClassSetView',
    component: DhcpClientClassSetViewComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<DhcpClientClassSetViewComponent>

export const SomeClasses: Story = {
    args: {
        clientClasses: ['access-point', 'router', 'DROP', 'custom'],
    },
}

export const NoClasses: Story = {
    args: {
        clientClasses: [],
    },
}
