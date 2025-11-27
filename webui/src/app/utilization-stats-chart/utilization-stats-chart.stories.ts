import { Meta, StoryObj } from '@storybook/angular'
import { UtilizationStatsChartComponent } from './utilization-stats-chart.component'

export default {
    title: 'App/UtilizationStatsChart',
    component: UtilizationStatsChartComponent,
} as Meta

type Story = StoryObj<UtilizationStatsChartComponent>

export const AllDHCPv4StatsAvailable: Story = {
    args: {
        leaseType: 'na',
        network: {
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
        },
    },
}

export const AllDHCPv6StatsAvailable: Story = {
    args: {
        leaseType: 'na',
        network: {
            addrUtilization: 30,
            stats: {
                'total-nas': 240,
                'assigned-nas': 70,
                'declined-nas': 10,
            },
        },
    },
}

export const AllPrefixStatsAvailable: Story = {
    args: {
        leaseType: 'pd',
        network: {
            pdUtilization: 15,
            stats: {
                'total-pds': 6400,
                'assigned-pds': 800,
            },
        },
    },
}

export const NoUtilization: Story = {
    args: {
        leaseType: 'na',
        network: {
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
        },
    },
}

export const NoDetailedStats: Story = {
    args: {
        leaseType: 'na',
        network: {
            addrUtilization: 50,
            stats: {},
        },
    },
}

export const DeclinesExceedAssignedWithFree: Story = {
    args: {
        leaseType: 'na',
        network: {
            addrUtilization: 34,
            stats: {
                'total-addresses': 847,
                'assigned-addresses': 100,
                'declined-addresses': 570,
            },
        },
    },
}

export const DeclinesExceedAssignedNoFree: Story = {
    args: {
        leaseType: 'na',
        network: {
            addrUtilization: 34,
            stats: {
                'total-addresses': 847,
                'assigned-addresses': 315,
                'declined-addresses': 570,
            },
        },
    },
}
