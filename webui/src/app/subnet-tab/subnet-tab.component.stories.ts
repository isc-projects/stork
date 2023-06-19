import { moduleMetadata, Meta, Story } from '@storybook/angular'
import { SubnetTabComponent } from './subnet-tab.component'
import { ChartModule } from 'primeng/chart'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { NumberPipe } from '../pipes/number.pipe'
import { FieldsetModule } from 'primeng/fieldset'
import { DividerModule } from 'primeng/divider'
import { TableModule } from 'primeng/table'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { RouterTestingModule } from '@angular/router/testing'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'

export default {
    title: 'App/SubnetTab',
    component: SubnetTabComponent,
    decorators: [
        moduleMetadata({
            imports: [
                ChartModule,
                DividerModule,
                FieldsetModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterTestingModule,
                TableModule,
                TooltipModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                DelegatedPrefixBarComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                NumberPipe,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
            ],
            providers: [],
        }),
    ],
} as Meta

const Template: Story<SubnetTabComponent> = (args: SubnetTabComponent) => ({
    props: args,
})

export const Subnet4 = Template.bind({})
Subnet4.args = {
    leaseType: 'address',
    subnet: {
        subnet: '192.0.2.0/24',
        sharedNetwork: 'Fiber',
        addrUtilization: 30,
        stats: {
            'total-addresses': 240,
            'assigned-addresses': 70,
            'declined-addresses': 10,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@192.0.2.1',
                pools: ['192.0.2.1-192.0.2.100'],
            },
        ],
    },
}

export const Subnet4NoPools = Template.bind({})
Subnet4NoPools.args = {
    leaseType: 'address',
    subnet: {
        subnet: '192.0.2.0/24',
        sharedNetwork: 'Fiber',
        addrUtilization: 50,
        stats: {
            'total-addresses': 30,
            'assigned-addresses': 15,
            'declined-addresses': 0,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@192.0.2.1',
                pools: null,
            },
        ],
    },
}

export const Subnet4NoPoolsInOneServer = Template.bind({})
Subnet4NoPoolsInOneServer.args = {
    leaseType: 'address',
    subnet: {
        subnet: '192.0.2.0/24',
        sharedNetwork: 'Fiber',
        addrUtilization: 30,
        stats: {
            'total-addresses': 240,
            'assigned-addresses': 70,
            'declined-addresses': 10,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@192.0.2.1',
                pools: null,
            },
            {
                id: 2,
                appName: 'bar@192.0.2.2',
                pools: ['192.0.2.10-192.0.2.20'],
            },
        ],
    },
}

export const Subnet6Address = Template.bind({})
Subnet6Address.args = {
    leaseType: 'na',
    subnet: {
        subnet: '2001:db8:1::/64',
        addrUtilization: 60,
        stats: {
            'total-nas': 1000,
            'assigned-nas': 30,
            'declined-nas': 10,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                appName: 'foo@2001:db8:1::1',
                pools: ['2001:db8:1::2-2001:db8:1::786'],
            },
        ],
    },
}

export const Subnet6Prefix = Template.bind({})
Subnet6Prefix.args = {
    leaseType: 'na',
    subnet: {
        subnet: '2001:db8:1::/64',
        pdUtilization: 60,
        stats: {
            'total-pds': 500,
            'assigned-pds': 358,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@2001:db8:1::1',
                prefixDelegationPools: [
                    {
                        prefix: '3000::',
                        delegatedLength: 80,
                    },
                ],
            },
        ],
    },
}

export const Subnet6AddressPrefix = Template.bind({})
Subnet6AddressPrefix.args = {
    leaseType: 'na',
    subnet: {
        subnet: '2001:db8:1::/64',
        addrUtilization: 88,
        pdUtilization: 60,
        stats: {
            'total-nas': 1024,
            'assigned-nas': 980,
            'declined-nas': 10,
            'total-pds': 500,
            'assigned-pds': 358,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@2001:db8:1::1',
                pools: ['2001:db8:1::2-2001:db8:1::768'],
                prefixDelegationPools: [
                    {
                        prefix: '3000::',
                        delegatedLength: 80,
                    },
                ],
            },
        ],
    },
}

export const Subnet6DifferentPoolsOnDifferentServers = Template.bind({})
Subnet6DifferentPoolsOnDifferentServers.args = {
    leaseType: 'na',
    subnet: {
        subnet: '2001:db8:1::/64',
        addrUtilization: 88,
        pdUtilization: 60,
        stats: {
            'total-nas': 1024,
            'assigned-nas': 980,
            'declined-nas': 10,
            'total-pds': 500,
            'assigned-pds': 358,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@2001:db8:1::1',
                pools: ['2001:db8:1::2-2001:db8:1::768'],
                prefixDelegationPools: [
                    {
                        prefix: '3000::',
                        delegatedLength: 80,
                    },
                ],
                stats: {
                    'total-nas': 1024,
                    'assigned-nas': 500,
                    'declined-nas': 4,
                    'total-pds': 300,
                    'assigned-pds': 200,
                },
            },
            {
                id: 2,
                appName: 'bar@2001:db8:2::5',
                pools: ['2001:db8:1::2-2001:db8:1::768'],
                prefixDelegationPools: [
                    {
                        prefix: '3000::',
                        delegatedLength: 80,
                    },
                    {
                        prefix: '3000:1::',
                        delegatedLength: 96,
                    },
                ],
                stats: {
                    'total-nas': 1024,
                    'assigned-nas': 480,
                    'declined-nas': 6,
                    'total-pds': 200,
                    'assigned-pds': 158,
                },
            },
        ],
    },
}

export const Subnet6NoPools = Template.bind({})
Subnet6NoPools.args = {
    leaseType: 'na',
    subnet: {
        subnet: '2001:db8:1::/64',
        addrUtilization: 88,
        pdUtilization: 60,
        stats: {
            'total-nas': 1024,
            'assigned-nas': 980,
            'declined-nas': 10,
            'total-pds': 500,
            'assigned-pds': 358,
        },
        statsCollectedAt: '2023-06-05',
        localSubnets: [
            {
                id: 1,
                appName: 'foo@2001:db8:1::1',
            },
        ],
    },
}
