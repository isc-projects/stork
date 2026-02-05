import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { HaStatusComponent } from './ha-status.component'
import { ServicesService, ServicesStatus } from '../backend'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { toastDecorator } from '../utils-stories'
import { provideRouter, withHashLocation } from '@angular/router'

let mockHubAndSpokeStatus: ServicesStatus = {
    items: [
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server1',
                    primaryServer: {
                        age: 0,
                        label: 'DHCPv4@localhost',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server1'],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                    secondaryServer: {
                        age: 0,
                        label: 'DHCPv4@remotehost',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'standby',
                        scopes: [],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                },
            },
        },
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server3',
                    primaryServer: {
                        age: 0,
                        label: 'DHCPv4@localhost',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server3'],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                    secondaryServer: {
                        age: 0,
                        label: 'DHCPv4@remotehost',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'standby',
                        scopes: [],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 1,
                        connectingClients: 5,
                        unackedClients: 1,
                        unackedClientsLeft: 4,
                        analyzedPackets: 0,
                    },
                },
            },
        },
    ],
}

let mockPassiveBackupStatus: ServicesStatus = {
    items: [
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server1',
                    primaryServer: {
                        age: 0,
                        label: 'DHCPv4@localhost',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server1'],
                        state: 'passive-backup',
                        statusTime: '2024-02-16',
                    },
                },
            },
        },
    ],
}

export default {
    title: 'App/HaStatus',
    component: HaStatusComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                ServicesService,
                MessageService,
                provideRouter([{ path: '**', component: HaStatusComponent }], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/123/services/status',
                method: 'GET',
                status: 200,
                delay: 200,
                response: mockHubAndSpokeStatus,
            },
            {
                url: 'http://localhost/api/daemons/234/services/status',
                method: 'GET',
                status: 200,
                delay: 200,
                response: mockPassiveBackupStatus,
            },
        ],
    },
} as Meta

type Story = StoryObj<HaStatusComponent>

export const hubAndSpoke: Story = {
    args: {
        daemonId: 123,
        daemonName: 'dhcp4',
    },
}

export const passiveBackup: Story = {
    args: {
        daemonId: 234,
        daemonName: 'dhcp4',
    },
}
