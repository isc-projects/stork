import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { SubnetFormComponent } from './subnet-form.component'
import { toastDecorator } from '../utils-stories'
import { MessageService } from 'primeng/api'
import { CreateSubnetBeginResponse, UpdateSubnetBeginResponse } from '../backend'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

let mockCreateSubnetBeginData: CreateSubnetBeginResponse = {
    id: 123,
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.6.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.8',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.8',
        },
    ],
    sharedNetworks4: [
        {
            id: 1,
            name: 'floor1',
            localSharedNetworks: [
                {
                    daemonId: 1,
                },
            ],
        },
        {
            id: 2,
            name: 'floor2',
            localSharedNetworks: [
                {
                    daemonId: 2,
                },
            ],
        },
        {
            id: 3,
            name: 'floor3',
            localSharedNetworks: [
                {
                    daemonId: 1,
                },
                {
                    daemonId: 2,
                },
            ],
        },
    ],
    sharedNetworks6: [],
    subnets: ['192.0.2.0/24', '192.0.3.0/24', '192.0.4.0/24', '2001:db8:1::/64', '2001:db8:2::/64'],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSubnet4BeginData: UpdateSubnetBeginResponse = {
    id: 123,
    subnet: {
        id: 123,
        subnet: '192.0.2.0/24',
        localSubnets: [
            {
                id: 123,
                appId: 234,
                daemonId: 1,
                appName: 'server 1',
                machineAddress: '10.1.1.1.',
                machineHostname: 'myhost.example.org',
                pools: [
                    {
                        pool: '192.0.2.10-192.0.2.100',
                        keaConfigPoolParameters: {
                            clientClass: 'foo',
                            requireClientClasses: ['foo', 'bar'],
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.10'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '',
                        },
                    },
                    {
                        pool: '192.0.2.200-192.0.2.250',
                    },
                ],
                keaConfigSubnetParameters: {
                    subnetLevelParameters: {
                        allocator: 'random',
                        options: [
                            {
                                alwaysSend: true,
                                code: 5,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.1'],
                                    },
                                ],
                                options: [],
                                universe: 4,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
            {
                id: 123,
                appId: 234,
                daemonId: 2,
                appName: 'server 2',
                machineAddress: '10.1.1.1.',
                machineHostname: 'myhost.example.org',
                pools: [
                    {
                        pool: '192.0.2.10-192.0.2.100',
                        keaConfigPoolParameters: {
                            clientClass: 'foo',
                            requireClientClasses: ['foo', 'bar'],
                            options: [],
                            optionsHash: '',
                        },
                    },
                ],
                keaConfigSubnetParameters: {
                    subnetLevelParameters: {
                        allocator: 'iterative',
                        options: [
                            {
                                alwaysSend: true,
                                code: 5,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.2'],
                                    },
                                ],
                                options: [],
                                universe: 4,
                            },
                        ],
                        optionsHash: '234',
                    },
                },
            },
        ],
    },
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.6.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.8',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.8',
        },
    ],
    sharedNetworks4: [
        {
            id: 1,
            name: 'floor1',
            localSharedNetworks: [
                {
                    daemonId: 1,
                },
            ],
        },
        {
            id: 2,
            name: 'floor2',
            localSharedNetworks: [
                {
                    daemonId: 2,
                },
            ],
        },
        {
            id: 3,
            name: 'floor3',
            localSharedNetworks: [
                {
                    daemonId: 1,
                },
                {
                    daemonId: 2,
                },
            ],
        },
    ],
    sharedNetworks6: [],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSubnet6BeginData: UpdateSubnetBeginResponse = {
    id: 123,
    subnet: {
        id: 234,
        subnet: '2001:db8:1::/64',
        localSubnets: [
            {
                id: 234,
                appId: 345,
                daemonId: 3,
                appName: 'server 1',
                machineAddress: '10.1.1.1',
                machineHostname: 'myhost.example.org',
                pools: [
                    {
                        pool: '2001:db8:1::10-2001:db8:1::100',
                        keaConfigPoolParameters: {
                            clientClass: 'foo',
                            requireClientClasses: ['foo', 'bar'],
                            options: [],
                            optionsHash: '',
                        },
                    },
                ],
                prefixDelegationPools: [
                    {
                        prefix: '3000:1::/16',
                        delegatedLength: 64,
                        excludedPrefix: null,
                    },
                ],
                keaConfigSubnetParameters: {
                    subnetLevelParameters: {
                        allocator: 'random',
                        options: [
                            {
                                alwaysSend: true,
                                code: 23,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv6-address',
                                        values: ['2001:db8:2::6789'],
                                    },
                                ],
                                options: [],
                                universe: 6,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
            {
                id: 345,
                appId: 456,
                daemonId: 4,
                appName: 'server 2',
                machineAddress: '10.1.1.1.',
                machineHostname: 'myhost.example.org',
                pools: [
                    {
                        pool: '2001:db8:1::10-2001:db8:1::100',
                        keaConfigPoolParameters: {
                            clientClass: 'foo',
                            requireClientClasses: ['foo', 'bar'],
                            options: [],
                            optionsHash: '',
                        },
                    },
                ],
                prefixDelegationPools: [
                    {
                        prefix: '3000:1::/16',
                        delegatedLength: 64,
                        excludedPrefix: null,
                    },
                ],
                keaConfigSubnetParameters: {
                    subnetLevelParameters: {
                        pdAllocator: 'iterative',
                        options: [
                            {
                                alwaysSend: true,
                                code: 23,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv6-address',
                                        values: ['2001:db8:2::6789'],
                                    },
                                ],
                                options: [],
                                universe: 6,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
        ],
    },
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.8',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.5.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.8',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.8',
        },
    ],
    sharedNetworks4: [],
    sharedNetworks6: [
        {
            id: 1,
            name: 'floor1',
            localSharedNetworks: [
                {
                    daemonId: 3,
                },
            ],
        },
        {
            id: 2,
            name: 'floor2',
            localSharedNetworks: [
                {
                    daemonId: 4,
                },
            ],
        },
        {
            id: 3,
            name: 'floor3',
            localSharedNetworks: [
                {
                    daemonId: 5,
                },
            ],
        },
    ],
    clientClasses: ['foo', 'bar'],
}

export default {
    title: 'App/SubnetForm',
    component: SubnetFormComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi())],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/subnets/new/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockCreateSubnetBeginData,
            },
            {
                url: 'http://localhost/subnets/123/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockUpdateSubnet4BeginData,
            },
            {
                url: 'http://localhost/subnets/234/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockUpdateSubnet6BeginData,
            },
            {
                url: 'http://localhost/subnets/345/transaction',
                method: 'POST',
                status: 400,
                delay: 2000,
                response: mockUpdateSubnet4BeginData,
            },
        ],
    },
} as Meta

type Story = StoryObj<SubnetFormComponent>

export const NewSubnet: Story = {
    args: {},
}

export const UpdatedSubnet4: Story = {
    args: {
        subnetId: 123,
    },
}

export const UpdatedSubnet6: Story = {
    args: {
        subnetId: 234,
    },
}

export const NoSubnetId: Story = {
    args: {},
}

export const ErrorMessage: Story = {
    args: {
        subnetId: 345,
    },
}
