import { ComponentFixture, TestBed } from '@angular/core/testing'

import { CommunicationStatusTreeComponent } from './communication-status-tree.component'
import { TreeModule } from 'primeng/tree'

describe('CommunicationStatusTreeComponent', () => {
    let component: CommunicationStatusTreeComponent
    let fixture: ComponentFixture<CommunicationStatusTreeComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TreeModule],
            declarations: [CommunicationStatusTreeComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(CommunicationStatusTreeComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert apps to tree', () => {
        component.apps = [
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
        ]
        component.ngOnInit()
        expect(component.nodes.length).toBe(6)

        // Stork agent on agent1.
        expect(component.nodes[0].icon).toBe('pi pi-server')
        expect(component.nodes[0].type).toBe('machine')
        expect(component.nodes[0].children.length).toBe(3)
        expect(component.nodes[0].styleClass).toBe('communication-failing')
        expect(component.nodes[0].expanded).toBeTrue()
        expect(component.nodes[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[0].data?.attrs?.address).toBe('agent1')

        // Kea agent on agent1.
        expect(component.nodes[0].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[0].children[0].type).toBe('kea')
        expect(component.nodes[0].children[0].children.length).toBe(1)
        expect(component.nodes[0].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[0].expanded).toBeTrue()
        expect(component.nodes[0].children[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[0].children[0].data?.attrs?.type).toBe('kea')
        expect(component.nodes[0].children[0].data?.attrs?.name).toBe('kea&bind9@agent1')

        // DHCPv4 server on agent1.
        expect(component.nodes[0].children[0].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[0].children[0].children[0].type).toBe('kea-daemon')
        expect(component.nodes[0].children[0].children[0].children).toBeFalsy()
        expect(component.nodes[0].children[0].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[0].children[0].data?.attrs?.id).toBe(3)
        expect(component.nodes[0].children[0].children[0].data?.attrs?.appType).toBe('kea')
        expect(component.nodes[0].children[0].children[0].data?.attrs?.appId).toBe(1)
        expect(component.nodes[0].children[0].children[0].data?.attrs?.name).toBe('DHCPv4')

        // named control channel on agent1.
        expect(component.nodes[0].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[0].children[1].type).toBe('named-channel')
        expect(component.nodes[0].children[1].children).toBeFalsy()
        expect(component.nodes[0].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[1].data?.attrs?.id).toBe(6)
        expect(component.nodes[0].children[1].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[0].children[1].data?.attrs?.appId).toBe(7)
        expect(component.nodes[0].children[1].data?.attrs?.name).toBe('named')
        expect(component.nodes[0].children[1].data?.channelName).toBe('Control')

        // named stats channel on agent1.
        expect(component.nodes[0].children[2].icon).toBe('pi pi-link')
        expect(component.nodes[0].children[2].type).toBe('named-channel')
        expect(component.nodes[0].children[2].children).toBeFalsy()
        expect(component.nodes[0].children[2].styleClass).toBe('communication-failing')
        expect(component.nodes[0].children[2].data?.attrs?.id).toBe(6)
        expect(component.nodes[0].children[2].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[0].children[2].data?.attrs?.appId).toBe(7)
        expect(component.nodes[0].children[2].data?.attrs?.name).toBe('named')
        expect(component.nodes[0].children[2].data?.channelName).toBe('Statistics')

        // Stork agent on agent2.
        expect(component.nodes[1].icon).toBe('pi pi-server')
        expect(component.nodes[1].type).toBe('machine')
        expect(component.nodes[1].children.length).toBe(1)
        expect(component.nodes[1].styleClass).toBe('communication-ok')
        expect(component.nodes[1].expanded).toBeTrue()
        expect(component.nodes[1].data?.attrs?.id).toBe(2)
        expect(component.nodes[1].data?.attrs?.address).toBe('agent2')

        // Kea agent on agent2.
        expect(component.nodes[1].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[1].children[0].type).toBe('kea')
        expect(component.nodes[1].children[0].children.length).toBe(3)
        expect(component.nodes[1].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[1].children[0].expanded).toBeTrue()
        expect(component.nodes[1].children[0].data?.attrs?.id).toBe(2)
        expect(component.nodes[1].children[0].data?.attrs?.type).toBe('kea')
        expect(component.nodes[1].children[0].data?.attrs?.name).toBe('kea@agent2')

        // DDNS server on agent2.
        expect(component.nodes[1].children[0].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[1].children[0].children[0].type).toBe('kea-daemon')
        expect(component.nodes[1].children[0].children[0].children).toBeFalsy()
        expect(component.nodes[1].children[0].children[0].styleClass).toBe('communication-disabled')
        expect(component.nodes[1].children[0].children[0].data?.attrs?.id).toBe(2)
        expect(component.nodes[1].children[0].children[0].data?.attrs?.appType).toBe('kea')
        expect(component.nodes[1].children[0].children[0].data?.attrs?.appId).toBe(2)
        expect(component.nodes[1].children[0].children[0].data?.attrs?.name).toBe('DDNS')

        // DHCPv4 server on agent2.
        expect(component.nodes[1].children[0].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[1].children[0].children[1].type).toBe('kea-daemon')
        expect(component.nodes[1].children[0].children[1].children).toBeFalsy()
        expect(component.nodes[1].children[0].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[1].children[0].children[1].data?.attrs?.id).toBe(3)
        expect(component.nodes[1].children[0].children[1].data?.attrs?.appType).toBe('kea')
        expect(component.nodes[1].children[0].children[1].data?.attrs?.appId).toBe(2)
        expect(component.nodes[1].children[0].children[1].data?.attrs?.name).toBe('DHCPv4')

        // DHCPv6 server on agent2.
        expect(component.nodes[1].children[0].children[2].icon).toBe('pi pi-link')
        expect(component.nodes[1].children[0].children[2].type).toBe('kea-daemon')
        expect(component.nodes[1].children[0].children[2].children).toBeFalsy()
        expect(component.nodes[1].children[0].children[2].styleClass).toBe('communication-disabled')
        expect(component.nodes[1].children[0].children[2].data?.attrs?.id).toBe(4)
        expect(component.nodes[1].children[0].children[2].data?.attrs?.appType).toBe('kea')
        expect(component.nodes[1].children[0].children[2].data?.attrs?.appId).toBe(2)
        expect(component.nodes[1].children[0].children[2].data?.attrs?.name).toBe('DHCPv6')

        // Stork agent on agent3.
        expect(component.nodes[2].icon).toBe('pi pi-server')
        expect(component.nodes[2].type).toBe('machine')
        expect(component.nodes[2].children.length).toBe(1)
        expect(component.nodes[2].styleClass).toBe('communication-ok')
        expect(component.nodes[2].expanded).toBeTrue()
        expect(component.nodes[2].data?.attrs?.id).toBe(3)
        expect(component.nodes[2].data?.attrs?.address).toBe('agent3')

        // Kea agent on agent3.
        expect(component.nodes[2].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[2].children[0].type).toBe('kea')
        expect(component.nodes[2].children[0].children.length).toBe(1)
        expect(component.nodes[2].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[2].children[0].expanded).toBeTrue()
        expect(component.nodes[2].children[0].data?.attrs?.id).toBe(3)
        expect(component.nodes[2].children[0].data?.attrs?.type).toBe('kea')
        expect(component.nodes[2].children[0].data?.attrs?.name).toBe('kea@agent3')

        // Stork agent on agent4.
        expect(component.nodes[3].icon).toBe('pi pi-server')
        expect(component.nodes[3].type).toBe('machine')
        expect(component.nodes[3].children.length).toBe(1)
        expect(component.nodes[3].styleClass).toBe('communication-failing')
        expect(component.nodes[3].expanded).toBeTrue()
        expect(component.nodes[3].data?.attrs?.id).toBe(4)
        expect(component.nodes[3].data?.attrs?.address).toBe('agent4')

        // Kea agent on agent4.
        expect(component.nodes[3].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[3].children[0].type).toBe('kea')
        expect(component.nodes[3].children[0].children.length).toBe(1)
        expect(component.nodes[3].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[3].children[0].expanded).toBeTrue()
        expect(component.nodes[3].children[0].data?.attrs?.id).toBe(4)
        expect(component.nodes[3].children[0].data?.attrs?.type).toBe('kea')
        expect(component.nodes[3].children[0].data?.attrs?.name).toBe('kea@agent4')

        // DHCPv4 server on agent4.
        expect(component.nodes[3].children[0].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[3].children[0].children[0].type).toBe('kea-daemon')
        expect(component.nodes[3].children[0].children[0].children).toBeFalsy()
        expect(component.nodes[3].children[0].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[3].children[0].children[0].data?.attrs?.id).toBe(3)
        expect(component.nodes[3].children[0].children[0].data?.attrs?.appType).toBe('kea')
        expect(component.nodes[3].children[0].children[0].data?.attrs?.appId).toBe(4)
        expect(component.nodes[3].children[0].children[0].data?.attrs?.name).toBe('DHCPv4')

        // Stork agent on agent5.
        expect(component.nodes[4].icon).toBe('pi pi-server')
        expect(component.nodes[4].type).toBe('machine')
        expect(component.nodes[4].children.length).toBe(2)
        expect(component.nodes[4].styleClass).toBe('communication-failing')
        expect(component.nodes[4].expanded).toBeTrue()
        expect(component.nodes[4].data?.attrs?.id).toBe(5)
        expect(component.nodes[4].data?.attrs?.address).toBe('agent5')

        // named control channel on agent5.
        expect(component.nodes[4].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[4].children[0].type).toBe('named-channel')
        expect(component.nodes[4].children[0].children).toBeFalsy()
        expect(component.nodes[4].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[4].children[0].data?.attrs?.id).toBe(6)
        expect(component.nodes[4].children[0].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[4].children[0].data?.attrs?.appId).toBe(5)
        expect(component.nodes[4].children[0].data?.attrs?.name).toBe('named')
        expect(component.nodes[4].children[0].data?.channelName).toBe('Control')

        // named statistics channel on agent5.
        expect(component.nodes[4].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[4].children[1].type).toBe('named-channel')
        expect(component.nodes[4].children[1].children).toBeFalsy()
        expect(component.nodes[4].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[4].children[1].data?.attrs?.id).toBe(6)
        expect(component.nodes[4].children[1].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[4].children[1].data?.attrs?.appId).toBe(5)
        expect(component.nodes[4].children[1].data?.attrs?.name).toBe('named')
        expect(component.nodes[4].children[1].data?.channelName).toBe('Statistics')

        // Stork agent on agent6.
        expect(component.nodes[5].icon).toBe('pi pi-server')
        expect(component.nodes[5].type).toBe('machine')
        expect(component.nodes[5].children.length).toBe(2)
        expect(component.nodes[5].styleClass).toBe('communication-ok')
        expect(component.nodes[5].expanded).toBeTrue()
        expect(component.nodes[5].data?.attrs?.id).toBe(6)
        expect(component.nodes[5].data?.attrs?.address).toBe('agent6')

        // named control channel on agent6.
        expect(component.nodes[5].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[5].children[0].type).toBe('named-channel')
        expect(component.nodes[5].children[0].children).toBeFalsy()
        expect(component.nodes[5].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[5].children[0].data?.attrs?.id).toBe(6)
        expect(component.nodes[5].children[0].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[5].children[0].data?.attrs?.appId).toBe(6)
        expect(component.nodes[5].children[0].data?.attrs?.name).toBe('named')
        expect(component.nodes[5].children[0].data?.channelName).toBe('Control')

        // named statistics channel on agent6.
        expect(component.nodes[5].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[5].children[1].type).toBe('named-channel')
        expect(component.nodes[5].children[1].children).toBeFalsy()
        expect(component.nodes[5].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[5].children[1].data?.attrs?.id).toBe(6)
        expect(component.nodes[5].children[1].data?.attrs?.appType).toBe('bind9')
        expect(component.nodes[5].children[1].data?.attrs?.appId).toBe(6)
        expect(component.nodes[5].children[1].data?.attrs?.name).toBe('named')
        expect(component.nodes[5].children[1].data?.channelName).toBe('Statistics')
    })
})
