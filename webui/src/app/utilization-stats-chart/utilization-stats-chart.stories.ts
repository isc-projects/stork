import { moduleMetadata, Meta, Story } from '@storybook/angular'
import { ChartModule } from 'primeng/chart'
import { UtilizationStatsChartComponent } from './utilization-stats-chart.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { LocalNumberPipe } from '../pipes/local-number.pipe'

export default {
    title: 'App/UtilizationStatsChart',
    component: UtilizationStatsChartComponent,
    decorators: [
        moduleMetadata({
            imports: [ChartModule, TooltipModule],
            declarations: [HumanCountComponent, HumanCountPipe, LocalNumberPipe],
            providers: [],
        }),
    ],
} as Meta

const Template: Story<UtilizationStatsChartComponent> = (args: UtilizationStatsChartComponent) => ({
    props: args,
})

export const AllDHCPv4StatsAvailable = Template.bind({})
AllDHCPv4StatsAvailable.args = {
    leaseType: 'address',
    network: {
        addrUtilization: 30,
        stats: {
            'total-addresses': 240,
            'assigned-addresses': 70,
            'declined-addresses': 10,
        },
    },
}

export const AllDHCPv6StatsAvailable = Template.bind({})
AllDHCPv6StatsAvailable.args = {
    leaseType: 'na',
    network: {
        addrUtilization: 30,
        stats: {
            'total-nas': 240,
            'assigned-nas': 70,
            'declined-nas': 10,
        },
    },
}

export const AllPrefixStatsAvailable = Template.bind({})
AllPrefixStatsAvailable.args = {
    leaseType: 'pd',
    network: {
        pdUtilization: 15,
        stats: {
            'total-pds': 6400,
            'assigned-pds': 800,
        },
    },
}

export const NoUtilization = Template.bind({})
NoUtilization.args = {
    leaseType: 'address',
    network: {
        stats: {
            'total-addresses': 240,
            'assigned-addresses': 70,
            'declined-addresses': 10,
        },
    },
}

export const NoDetailedStats = Template.bind({})
NoDetailedStats.args = {
    leaseType: 'address',
    network: {
        addrUtilization: 50,
        stats: {},
    },
}
