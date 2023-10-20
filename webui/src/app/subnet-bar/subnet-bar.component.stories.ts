import { Meta, Story, applicationConfig, moduleMetadata } from '@storybook/angular'
import { SubnetBarComponent } from './subnet-bar.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { Subnet } from '../backend'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'

export default {
    title: 'App/SubnetBar',
    component: SubnetBarComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [RouterTestingModule, TooltipModule],
            declarations: [EntityLinkComponent],
        }),
    ],
} as Meta

const Template: Story<SubnetBarComponent> = (args: SubnetBarComponent) => ({
    props: args,
})

export const ipv4NoStats = Template.bind({})
ipv4NoStats.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
    } as Subnet,
}

export const ipv4NoStatsUtilization = Template.bind({})
ipv4NoStatsUtilization.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        addrUtilization: 86,
    } as Subnet,
}

export const ipv4Stats = Template.bind({})
ipv4Stats.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        stats: {
            'total-addresses': 50,
            'assigned-addresses': 20,
            'declined-addresses': 5,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv4UtilizationLow = Template.bind({})
ipv4UtilizationLow.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        addrUtilization: 30,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 30,
            'declined-addresses': 0,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv4UtilizationMedium = Template.bind({})
ipv4UtilizationMedium.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        addrUtilization: 85,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 85,
            'declined-addresses': 0,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv4UtilizationHigh = Template.bind({})
ipv4UtilizationHigh.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        addrUtilization: 95,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 95,
            'declined-addresses': 0,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv4UtilizationExceed = Template.bind({})
ipv4UtilizationExceed.args = {
    subnet: {
        id: 42,
        subnet: '42.42.0.0/16',
        addrUtilization: 110,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 110,
            'declined-addresses': 0,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6NoStats = Template.bind({})
ipv6NoStats.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
    } as Subnet,
}

export const ipv6NoStatsUtilization = Template.bind({})
ipv6NoStatsUtilization.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        addrUtilization: 85,
    } as Subnet,
}

export const ipv6Stats = Template.bind({})
ipv6Stats.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 50,
            'assigned-nas': 20,
            'declined-nas': 5,
            'total-pds': 70,
            'assigned-pds': 30,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6StatsLongPrefix = Template.bind({})
ipv6StatsLongPrefix.args = {
    subnet: {
        id: 42,
        subnet: '3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/125',
        stats: {
            'total-nas': 4,
            'assigned-nas': 3,
            'declined-nas': 1,
            'total-pds': 0,
            'assigned-pds': 0,
        },
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressLow = Template.bind({})
ipv6UtilizationAddressLow.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 20,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 80,
        },
        addrUtilization: 20,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressMedium = Template.bind({})
ipv6UtilizationAddressMedium.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 162,
        },
        addrUtilization: 85,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressHigh = Template.bind({})
ipv6UtilizationAddressHigh.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 95,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 182,
        },
        addrUtilization: 95,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressExceed = Template.bind({})
ipv6UtilizationAddressExceed.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 110,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 250,
        },
        addrUtilization: 110,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationDelegatedPrefixLow = Template.bind({})
ipv6UtilizationDelegatedPrefixLow.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 20,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 80,
        },
        pdUtilization: 40,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationDelegatedPrefixMedium = Template.bind({})
ipv6UtilizationDelegatedPrefixMedium.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 162,
        },
        pdUtilization: 81,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationDelegatedPrefixHigh = Template.bind({})
ipv6UtilizationDelegatedPrefixHigh.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 95,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 182,
        },
        pdUtilization: 91,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationDelegatedPrefixExceed = Template.bind({})
ipv6UtilizationDelegatedPrefixExceed.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 110,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 250,
        },
        pdUtilization: 125,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressMediumDelegatedPrefixHigh = Template.bind({})
ipv6UtilizationAddressMediumDelegatedPrefixHigh.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 190,
        },
        addrUtilization: 85,
        pdUtilization: 95,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationAddressMediumDelegatedPrefixMedium = Template.bind({})
ipv6UtilizationAddressMediumDelegatedPrefixMedium.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 170,
        },
        addrUtilization: 85,
        pdUtilization: 85,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationNoDelegatedPrefixes = Template.bind({})
ipv6UtilizationNoDelegatedPrefixes.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 0,
            'assigned-pds': 0,
        },
        addrUtilization: 85,
        pdUtilization: 0,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationNoAddresses = Template.bind({})
ipv6UtilizationNoAddresses.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 0,
            'assigned-nas': 0,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 170,
        },
        addrUtilization: 0,
        pdUtilization: 85,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}

export const ipv6UtilizationNoAddressesAndDelegatedPrefixes = Template.bind({})
ipv6UtilizationNoAddressesAndDelegatedPrefixes.args = {
    subnet: {
        id: 42,
        subnet: '3001:1::/64',
        stats: {
            'total-nas': 0,
            'assigned-nas': 0,
            'declined-nas': 0,
            'total-pds': 0,
            'assigned-pds': 0,
        },
        addrUtilization: 0,
        pdUtilization: 0,
        statsCollectedAt: '2022-12-28T14:59:00',
    } as Subnet,
}
