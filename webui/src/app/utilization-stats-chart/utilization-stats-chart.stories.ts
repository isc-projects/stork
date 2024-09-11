import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { ChartModule } from 'primeng/chart'
import { UtilizationStatsChartComponent } from './utilization-stats-chart.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { PositivePipe } from '../pipes/positive.pipe'

export default {
    title: 'App/UtilizationStatsChart',
    component: UtilizationStatsChartComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [ChartModule, TooltipModule],
            declarations: [HumanCountComponent, HumanCountPipe, LocalNumberPipe, PositivePipe],
        }),
    ],
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
