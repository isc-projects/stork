import { Meta, StoryObj } from '@storybook/angular'
import { OutOfPoolBarComponent } from './out-of-pool-bar.component'

export default {
    title: 'App/OutOfPoolBar',
    component: OutOfPoolBarComponent,
} as Meta

type Story = StoryObj<OutOfPoolBarComponent>

export const IPv4: Story = {
    args: {
        utilization: 25,
        stats: {
            'total-addresses': 1000,
            'assigned-addresses': 500,
            'total-out-of-pool-addresses': 100,
            'assigned-out-of-pool-addresses': 50,
            'declined-out-of-pool-addresses': 10,
        },
        statsCollectedAt: '2023-10-01T12:00:00Z',
    },
}

export const IPv6: Story = {
    args: {
        utilization: 60,
        stats: {
            'total-pds': 1000,
            'assigned-pds': 500,
            'total-nas': 1000,
            'assigned-nas': 500,
            'total-out-of-pool-nas': 100,
            'assigned-out-of-pool-nas': 50,
            'declined-out-of-pool-nas': 20,
            'total-out-of-pool-pds': 100,
            'assigned-out-of-pool-pds': 50,
        },
        statsCollectedAt: '2023-10-01T12:00:00Z',
    },
}

export const IPv6PD: Story = {
    args: {
        utilization: 75,
        isPD: true,
        stats: {
            'total-pds': 1000,
            'assigned-pds': 500,
            'total-out-of-pool-pds': 100,
            'assigned-out-of-pool-pds': 50,
            'declined-out-of-pool-pds': 15,
        },
        statsCollectedAt: '2023-10-01T12:00:00Z',
    },
}
