import { Meta, StoryObj, applicationConfig, moduleMetadata } from '@storybook/angular'
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

type Story = StoryObj<SubnetBarComponent>

export const ipv4NoStats: Story = {
    args: {
        subnet: {
            id: 42,
            subnet: '42.42.0.0/16',
        } as Subnet,
    },
}

export const ipv4NoStatsUtilization: Story = {
    args: {
        subnet: {
            id: 42,
            subnet: '42.42.0.0/16',
            addrUtilization: 86,
        } as Subnet,
    },
}

export const ipv4Stats: Story = {
    args: {
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
    },
}

export const ipv4UtilizationLow: Story = {
    args: {
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
    },
}

export const ipv4UtilizationMedium: Story = {
    args: {
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
    },
}

export const ipv4UtilizationHigh: Story = {
    args: {
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
    },
}

export const ipv4UtilizationExceed: Story = {
    args: {
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
    },
}

export const ipv6NoStats: Story = {
    args: {
        subnet: {
            id: 42,
            subnet: '3001:1::/64',
        } as Subnet,
    },
}

export const ipv6NoStatsUtilization: Story = {
    args: {
        subnet: {
            id: 42,
            subnet: '3001:1::/64',
            addrUtilization: 85,
        } as Subnet,
    },
}

export const ipv6Stats: Story = {
    args: {
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
    },
}

export const ipv6StatsLongPrefix: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressLow: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressMedium: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressHigh: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressExceed: Story = {
    args: {
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
    },
}

export const ipv6UtilizationDelegatedPrefixLow: Story = {
    args: {
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
    },
}

export const ipv6UtilizationDelegatedPrefixMedium: Story = {
    args: {
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
    },
}

export const ipv6UtilizationDelegatedPrefixHigh: Story = {
    args: {
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
    },
}

export const ipv6UtilizationDelegatedPrefixExceed: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressMediumDelegatedPrefixHigh: Story = {
    args: {
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
    },
}

export const ipv6UtilizationAddressMediumDelegatedPrefixMedium: Story = {
    args: {
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
    },
}

export const ipv6UtilizationNoDelegatedPrefixes: Story = {
    args: {
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
    },
}

export const ipv6UtilizationNoAddresses: Story = {
    args: {
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
    },
}

export const ipv6UtilizationNoAddressesAndDelegatedPrefixes: Story = {
    args: {
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
    },
}
