import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { CommunicationStatusTreeComponent } from './communication-status-tree.component'
import { provideRouter, withHashLocation } from '@angular/router'

export default {
    title: 'App/CommunicationStatusTree',
    component: CommunicationStatusTreeComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideRouter([{ path: '**', component: CommunicationStatusTreeComponent }], withHashLocation()),
            ],
        }),
    ],
} as Meta

type Story = StoryObj<CommunicationStatusTreeComponent>

export const IssuesTree: Story = {
    args: {
        daemons: [
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 1,
                machineLabel: 'agent1',
                active: true,
                agentCommErrors: 1,
                id: 1,
                monitored: true,
                name: 'ca',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 1,
                machineLabel: 'agent1',
                active: true,
                agentCommErrors: 0,
                id: 2,
                monitored: true,
                name: 'dhcp4',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 2,
                machineLabel: 'agent2',
                daemonCommErrors: 3,
                active: true,
                id: 3,
                monitored: true,
                name: 'ca',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 2,
                machineLabel: 'agent2',
                daemonCommErrors: 2,
                id: 4,
                name: 'd2',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 2,
                machineLabel: 'agent2',
                active: true,
                id: 5,
                monitored: true,
                name: 'dhcp4',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 2,
                machineLabel: 'agent2',
                daemonCommErrors: 3,
                id: 6,
                name: 'dhcp6',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 3,
                machineLabel: 'agent3',
                active: true,
                caCommErrors: 1,
                id: 7,
                monitored: true,
                name: 'ca',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 3,
                machineLabel: 'agent3',
                active: true,
                agentCommErrors: 0,
                id: 8,
                monitored: true,
                name: 'dhcp4',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 4,
                machineLabel: 'agent4',
                active: true,
                caCommErrors: 1,
                id: 9,
                monitored: true,
                name: 'ca',
            },
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                machineId: 4,
                machineLabel: 'agent4',
                active: true,
                agentCommErrors: 5,
                daemonCommErrors: 4,
                id: 10,
                monitored: true,
                name: 'dhcp4',
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
                machineId: 5,
                machineLabel: 'agent5',
                active: true,
                id: 11,
                monitored: true,
                name: 'named',
                agentCommErrors: 5,
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
                machineId: 6,
                machineLabel: 'agent6',
                active: true,
                id: 12,
                monitored: true,
                name: 'named',
                daemonCommErrors: 4,
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
                machineId: 1,
                machineLabel: 'agent1',
                active: true,
                id: 13,
                monitored: true,
                name: 'named',
                statsCommErrors: 7,
            },
        ],
    },
}
