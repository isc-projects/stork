import { Meta, StoryObj } from '@storybook/angular'
import { AddressPoolBarComponent } from './address-pool-bar.component'

export default {
    title: 'App/AddressPoolBar',
    component: AddressPoolBarComponent,
} as Meta

type Story = StoryObj<AddressPoolBarComponent>

export const Range: Story = {
    args: {
        pool: {
            pool: '10.0.0.1-10.0.0.42',
            utilization: 25,
        },
    },
}

export const CIDR: Story = {
    args: {
        pool: {
            pool: '10.0.0.1/24',
            utilization: 60,
        },
    },
}
