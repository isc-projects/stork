import { Meta, StoryObj, applicationConfig, moduleMetadata } from '@storybook/angular'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { Subnet } from '../backend'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { UtilizationBarComponent } from './utilization-bar.component'

export default {
    title: 'App/UtilizationBar',
    component: UtilizationBarComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [RouterTestingModule, TooltipModule],
        }),
    ],
} as Meta

type Story = StoryObj<UtilizationBarComponent>

export const singleNoUtilization: Story = {
    args: {},
}

export const singleUtilizationLow: Story = {
    args: {
        utilizationPrimary: 30,
        kindPrimary: 'Addresses',
    },
}

export const singleUtilizationMedium: Story = {
    args: {
        utilizationPrimary: 85,
        kindPrimary: 'Addresses',
    },
}

export const singleUtilizationHigh: Story = {
    args: {
        utilizationPrimary: 95,
        kindPrimary: 'Addresses',
    },
}

export const singleUtilizationExceed: Story = {
    args: {
        utilizationPrimary: 110,
        kindPrimary: 'Addresses',
    },
}

export const doubleNoUtilization: Story = {
    args: {},
}

// export const doubleLongLabel: Story = {
//     args: {
//         subnet: {
//             id: 42,
//             subnet: '3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/125',
//             stats: {
//                 'total-nas': 4,
//                 'assigned-nas': 3,
//                 'declined-nas': 1,
//                 'total-pds': 0,
//                 'assigned-pds': 0,
//             },
//             statsCollectedAt: '2022-12-28T14:59:00',
//         } as Subnet,
//     },
// }

export const doubleUtilizationPrimaryLow: Story = {
    args: {
        utilizationPrimary: 30,
        kindPrimary: 'Addresses',
        utilizationSecondary: null,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationPrimaryMedium: Story = {
    args: {
        utilizationPrimary: 85,
        kindPrimary: 'Addresses',
        utilizationSecondary: null,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationPrimaryHigh: Story = {
    args: {
        utilizationPrimary: 95,
        kindPrimary: 'Addresses',
        utilizationSecondary: null,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationPrimaryExceed: Story = {
    args: {
        utilizationPrimary: 110,
        kindPrimary: 'Addresses',
        utilizationSecondary: null,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationSecondaryLow: Story = {
    args: {
        utilizationPrimary: null,
        kindPrimary: 'Addresses',
        utilizationSecondary: 30,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationSecondaryMedium: Story = {
    args: {
        utilizationPrimary: null,
        kindPrimary: 'Addresses',
        utilizationSecondary: 85,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationSecondaryHigh: Story = {
    args: {
        utilizationPrimary: null,
        kindPrimary: 'Addresses',
        utilizationSecondary: 95,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationSecondaryExceed: Story = {
    args: {
        utilizationPrimary: null,
        kindPrimary: 'Addresses',
        utilizationSecondary: 110,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationPrimaryMediumSecondaryHigh: Story = {
    args: {
        utilizationPrimary: 85,
        kindPrimary: 'Addresses',
        utilizationSecondary: 95,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationPrimaryMediumSecondaryMedium: Story = {
    args: {
        utilizationPrimary: 85,
        kindPrimary: 'Addresses',
        utilizationSecondary: 85,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationZeroSecondary: Story = {
    args: {
        utilizationPrimary: 85,
        kindPrimary: 'Addresses',
        utilizationSecondary: 0,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationZeroPrimary: Story = {
    args: {
        utilizationPrimary: 0,
        kindPrimary: 'Addresses',
        utilizationSecondary: 85,
        kindSecondary: 'PD',
    },
}

export const doubleUtilizationZeroBoth: Story = {
    args: {
        utilizationPrimary: 0,
        kindPrimary: 'Addresses',
        utilizationSecondary: 0,
        kindSecondary: 'PD',
    },
}
