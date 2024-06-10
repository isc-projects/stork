import { Meta, StoryObj, applicationConfig, moduleMetadata } from '@storybook/angular'
import { CommunicationStatusTreeComponent } from './communication-status-tree.component'
import { TreeModule } from 'primeng/tree'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { RouterTestingModule } from '@angular/router/testing'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TooltipModule } from 'primeng/tooltip'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'

export default {
    title: 'App/CommunicationStatusTree',
    component: CommunicationStatusTreeComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [
                BreadcrumbModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterTestingModule,
                TooltipModule,
                ,
                TreeModule,
            ],
            declarations: [
                BreadcrumbsComponent,
                CommunicationStatusTreeComponent,
                EntityLinkComponent,
                HelpTipComponent,
            ],
        }),
    ],
} as Meta

type Story = StoryObj<CommunicationStatusTreeComponent>

export const IssuesTree: Story = {
    args: {
        apps: [
            // Kea app with the Communication issues with Stork Agent.
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                details: {
                    daemons: [
                        {
                            active: true,
                            agentCommErrors: 1,
                            id: 1,
                            monitored: true,
                            name: 'ca',
                        },
                        {
                            active: true,
                            agentCommErrors: 0,
                            id: 3,
                            monitored: true,
                            name: 'dhcp4',
                        },
                    ],
                },
                id: 1,
                machine: {
                    address: 'agent1',
                    hostname: 'agent1',
                    id: 1,
                },
                name: 'kea&bind9@agent1',
                type: 'kea',
            },
            // Kea app with the Communication errors with some daemons.
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                details: {
                    daemons: [
                        {
                            daemonCommErrors: 3,
                            active: true,
                            id: 1,
                            monitored: true,
                            name: 'ca',
                        },
                        {
                            daemonCommErrors: 2,
                            id: 2,
                            name: 'd2',
                        },
                        {
                            active: true,
                            id: 3,
                            monitored: true,
                            name: 'dhcp4',
                        },
                        {
                            daemonCommErrors: 3,
                            id: 4,
                            name: 'dhcp6',
                        },
                    ],
                },
                id: 2,
                machine: {
                    address: 'agent2',
                    hostname: 'agent2',
                    id: 2,
                },
                name: 'kea@agent2',
                type: 'kea',
            },
            // Kea app with the Communication issues with the Kea Control Agent.
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                details: {
                    daemons: [
                        {
                            active: true,
                            caCommErrors: 1,
                            id: 1,
                            monitored: true,
                            name: 'ca',
                        },
                        {
                            active: true,
                            agentCommErrors: 0,
                            id: 3,
                            monitored: true,
                            name: 'dhcp4',
                        },
                    ],
                },
                id: 3,
                machine: {
                    address: 'agent3',
                    hostname: 'agent3',
                    id: 3,
                },
                name: 'kea@agent3',
                type: 'kea',
            },
            // Kea app with the Communication issues at all levels.
            {
                accessPoints: [
                    {
                        address: '127.0.0.1',
                        port: 8000,
                        type: 'control',
                    },
                ],
                details: {
                    daemons: [
                        {
                            active: true,
                            caCommErrors: 1,
                            id: 1,
                            monitored: true,
                            name: 'ca',
                        },
                        {
                            active: true,
                            agentCommErrors: 5,
                            daemonCommErrors: 4,
                            id: 3,
                            monitored: true,
                            name: 'dhcp4',
                        },
                    ],
                },
                id: 4,
                machine: {
                    address: 'agent4',
                    hostname: 'agent4',
                    id: 4,
                },
                name: 'kea@agent4',
                type: 'kea',
            },
            // Bind9 app with the Communication issues with the Stork Agent.
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
                details: {
                    daemons: [],
                    daemon: {
                        active: true,
                        id: 6,
                        monitored: true,
                        name: 'named',
                        agentCommErrors: 5,
                    },
                },
                id: 5,
                machine: {
                    address: 'agent5',
                    hostname: 'agent5',
                    id: 5,
                },
                name: 'bind9@agent5',
                type: 'bind9',
            },
            // Bind9 app with the Communication issues over RNDC.
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
                details: {
                    daemons: [],
                    daemon: {
                        active: true,
                        id: 6,
                        monitored: true,
                        name: 'named',
                        rndcCommErrors: 4,
                    },
                },
                id: 6,
                machine: {
                    address: 'agent6',
                    hostname: 'agent6',
                    id: 6,
                },
                name: 'bind9@agent6',
                type: 'bind9',
            },
            // Bind9 app with the Communication issues over stats. It runs
            // on the same machine as first Kea.
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
                details: {
                    daemon: {
                        active: true,
                        id: 6,
                        monitored: true,
                        name: 'named',
                        statsCommErrors: 7,
                    },
                },
                id: 7,
                machine: {
                    address: 'agent1',
                    hostname: 'agent1',
                    id: 1,
                },
                name: 'kea&bind9@agent1',
                type: 'bind9',
            },
        ],
    },
}
