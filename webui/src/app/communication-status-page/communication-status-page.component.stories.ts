import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { CommunicationStatusPageComponent } from './communication-status-page.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { toastDecorator } from '../utils-stories'
import { MessageService } from 'primeng/api'
import { provideRouter, withHashLocation } from '@angular/router'
import { Daemons } from '../backend'

let mockGetDaemonsWithCommunicationIssues: Daemons = {
    items: [
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            agentCommErrors: 1,
            name: 'ca',
            id: 1,
            machineId: 1,
            machineLabel: 'agent1',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            agentCommErrors: 0,
            name: 'dhcp4',
            id: 3,
            machineId: 1,
            machineLabel: 'agent1',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            daemonCommErrors: 3,
            active: true,
            monitored: true,
            name: 'ca',
            id: 21,
            machineId: 2,
            machineLabel: 'agent2',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            daemonCommErrors: 2,
            name: 'd2',
            id: 22,
            machineId: 2,
            machineLabel: 'agent2',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            name: 'dhcp4',
            id: 23,
            machineId: 2,
            machineLabel: 'agent2',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            daemonCommErrors: 3,
            name: 'dhcp6',
            id: 24,
            machineId: 2,
            machineLabel: 'agent2',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            caCommErrors: 1,
            name: 'ca',
            id: 31,
            machineId: 3,
            machineLabel: 'agent3',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            agentCommErrors: 0,
            name: 'dhcp4',
            id: 33,
            machineId: 3,
            machineLabel: 'agent3',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            caCommErrors: 1,
            name: 'ca',
            id: 41,
            machineId: 4,
            machineLabel: 'agent4',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 8000,
                    type: 'control',
                },
            ],
            active: true,
            monitored: true,
            agentCommErrors: 5,
            daemonCommErrors: 4,
            name: 'dhcp4',
            id: 43,
            machineLabel: 'agent4',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 953,
                    type: 'control',
                },
                {
                    address: '127.0.0.1',
                    port: 8053,
                    type: 'statistics',
                },
            ],
            active: true,
            monitored: true,
            agentCommErrors: 5,
            name: 'named',
            id: 56,
            machineId: 5,
            machineLabel: 'agent5',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 953,
                    type: 'control',
                },
                {
                    address: '127.0.0.1',
                    port: 8053,
                    type: 'statistics',
                },
            ],
            active: true,
            monitored: true,
            daemonCommErrors: 4,
            name: 'named',
            id: 66,
            machineId: 6,
            machineLabel: 'agent6',
        },
        {
            accessPoints: [
                {
                    address: '127.0.0.1',
                    port: 953,
                    type: 'control',
                },
                {
                    address: '127.0.0.1',
                    port: 8053,
                    type: 'statistics',
                },
            ],
            active: true,
            monitored: true,
            statsCommErrors: 7,
            name: 'named',
            id: 76,
            machineId: 1,
            machineLabel: 'agent1',
        },
    ],
}

export default {
    title: 'App/CommunicationStatusPage',
    component: CommunicationStatusPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter([{ path: '**', component: CommunicationStatusPageComponent }], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/communication-issues',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: mockGetDaemonsWithCommunicationIssues,
            },
        ],
    },
} as Meta

type Story = StoryObj<CommunicationStatusPageComponent>

export const IssuesTree: Story = {
    args: {},
}
