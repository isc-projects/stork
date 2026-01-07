import { Meta, StoryObj } from '@storybook/angular'
import { CascadedParametersBoardComponent } from './cascaded-parameters-board.component'
import { KeaConfigSubnetDerivedParameters } from '../backend'

export default {
    title: 'App/CascadedParametersBoard',
    component: CascadedParametersBoardComponent,
} as Meta

type Story = StoryObj<CascadedParametersBoardComponent<KeaConfigSubnetDerivedParameters>>

export const SameParameters: Story = {
    args: {
        levels: ['Subnet', 'Shared Network', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'his',
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'example',
                        ddnsOverrideClientUpdate: true,
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        cacheThreshold: 0.22,
                        cacheMaxAge: 900,
                        clientClass: 'abc',
                        requireClientClasses: ['bar'],
                        ddnsGeneratedPrefix: 'hishost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.21,
                        cacheMaxAge: 800,
                        clientClass: 'ibi',
                        requireClientClasses: ['abc', 'dec'],
                        ddnsGeneratedPrefix: 'her',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['aaa'],
                        ddnsGeneratedPrefix: 'ours',
                        ddnsOverrideClientUpdate: false,
                    },
                ],
            },
        ],
    },
}

export const DistinctParameters: Story = {
    args: {
        levels: ['Subnet', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                    },
                    {
                        cacheMaxAge: 1000,
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        clientClass: 'abc',
                    },
                    {
                        requireClientClasses: ['abc', 'dec'],
                    },
                ],
            },
        ],
    },
}

export const ExcludedParameters: Story = {
    args: {
        levels: ['Subnet', 'Shared Network', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'his',
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'example',
                        ddnsOverrideClientUpdate: true,
                    },
                ],
            },
        ],
        excludedParameters: ['clientClass', 'ddnsOverrideClientUpdate'],
    },
}
