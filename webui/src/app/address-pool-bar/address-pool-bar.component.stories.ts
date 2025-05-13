import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { AddressPoolBarComponent } from './address-pool-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'

export default {
    title: 'App/AddressPoolBar',
    component: AddressPoolBarComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [TooltipModule],
            declarations: [UtilizationBarComponent],
        }),
    ],
} as Meta

type Story = StoryObj<AddressPoolBarComponent>

export const Primary: Story = {
    args: {
        pool: {
            pool: '10.0.0.1-10.0.0.42',
            utilization: 25,
        },
    },
}
