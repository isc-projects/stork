import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view.component'

import { Story, Meta, moduleMetadata } from '@storybook/angular'
import { ChipModule } from 'primeng/chip'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'

export default {
    title: 'App/DhcpClientClassSetView',
    component: DhcpClientClassSetViewComponent,
    decorators: [
        moduleMetadata({
            imports: [ChipModule, NoopAnimationsModule, ToastModule],
            providers: [MessageService],
        }),
        toastDecorator,
    ],
} as Meta

const Template: Story<DhcpClientClassSetViewComponent> = (args: DhcpClientClassSetViewComponent) => ({
    props: args,
})

export const SomeClasses = Template.bind({})
SomeClasses.args = {
    clientClasses: ['access-point', 'router', 'DROP', 'custom'],
}

export const NoClasses = Template.bind({})
NoClasses.args = {
    clientClasses: [],
}
