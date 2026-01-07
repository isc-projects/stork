import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { SharedNetworkFormComponent } from './shared-network-form.component'
import { toastDecorator } from '../utils-stories'
import { MessageService } from 'primeng/api'
import { CreateSharedNetworkBeginResponse, UpdateSharedNetworkBeginResponse } from '../backend'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter, withHashLocation } from '@angular/router'

let mockCreateSharedNetwork4BeginData: CreateSharedNetworkBeginResponse = {
    id: 123,
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.4.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: ['floor1', 'floor2', 'floor3', 'stanza'],
    sharedNetworks6: [],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSharedNetwork4BeginData: UpdateSharedNetworkBeginResponse = {
    id: 123,
    sharedNetwork: {
        id: 123,
        name: 'stanza',
        universe: 4,
        localSharedNetworks: [
            {
                appId: 234,
                daemonId: 1,
                appName: 'server 1',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
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
                appId: 234,
                daemonId: 2,
                appName: 'server 2',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
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
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.4.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: ['floor1', 'floor2', 'floor3', 'stanza'],
    sharedNetworks6: [],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSharedNetwork6BeginData: UpdateSharedNetworkBeginResponse = {
    id: 234,
    sharedNetwork: {
        id: 234,
        name: 'bella',
        universe: 6,
        localSharedNetworks: [
            {
                appId: 234,
                daemonId: 4,
                appName: 'server 1',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
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
                appId: 345,
                daemonId: 5,
                appName: 'server 2',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
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
        ],
    },
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: [],
    sharedNetworks6: ['floor1', 'floor2', 'floor3', 'bella'],
    clientClasses: ['foo', 'bar'],
}

export default {
    title: 'App/SharedNetworkForm',
    component: SharedNetworkFormComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter([{ path: '**', component: SharedNetworkFormComponent }], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/shared-networks/new/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockCreateSharedNetwork4BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/123/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockUpdateSharedNetwork4BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/234/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockUpdateSharedNetwork6BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/345/transaction',
                method: 'POST',
                status: 400,
                delay: 2000,
                response: mockUpdateSharedNetwork4BeginData,
            },
        ],
    },
} as Meta

type Story = StoryObj<SharedNetworkFormComponent>

export const NewSharedNetwork: Story = {
    args: {},
}

export const UpdatedSharedNetwork4: Story = {
    args: {
        sharedNetworkId: 123,
    },
}

export const UpdatedSharedNetwork6: Story = {
    args: {
        sharedNetworkId: 234,
    },
}

export const NoSharedNetworkId: Story = {
    args: {},
}

export const ErrorMessage: Story = {
    args: {
        sharedNetworkId: 345,
    },
}
