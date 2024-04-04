import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { ChipModule } from 'primeng/chip'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'

export default {
    title: 'App/DhcpClientClassSetView',
    component: DhcpClientClassSetViewComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService],
        }),
        moduleMetadata({
            imports: [ChipModule, NoopAnimationsModule, ToastModule],
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
