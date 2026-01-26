import { ComponentFixture, TestBed } from '@angular/core/testing'

import { CommunicationStatusTreeComponent } from './communication-status-tree.component'

describe('CommunicationStatusTreeComponent', () => {
    let component: CommunicationStatusTreeComponent
    let fixture: ComponentFixture<CommunicationStatusTreeComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(CommunicationStatusTreeComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert daemons to tree', () => {
        component.daemons = [
            // Kea daemons on agent1.
            {
                id: 1,
                name: 'ca',
                machineId: 1,
                machineLabel: 'agent1',
                active: true,
                monitored: true,
                agentCommErrors: 1,
            },
            {
                id: 3,
                name: 'dhcp4',
                machineId: 1,
                machineLabel: 'agent1',
                active: true,
                monitored: true,
                agentCommErrors: 0,
            },
            // Kea daemons on agent2.
            {
                id: 1,
                name: 'ca',
                machineId: 2,
                machineLabel: 'agent2',
                active: true,
                monitored: true,
            },
            {
                id: 2,
                name: 'd2',
                machineId: 2,
                machineLabel: 'agent2',
                daemonCommErrors: 2,
            },
            {
                id: 3,
                name: 'dhcp4',
                machineId: 2,
                machineLabel: 'agent2',
                active: true,
                monitored: true,
            },
            {
                id: 4,
                name: 'dhcp6',
                machineId: 2,
                machineLabel: 'agent2',
                daemonCommErrors: 3,
            },
            // Kea daemons on agent3.
            {
                id: 1,
                name: 'ca',
                machineId: 3,
                machineLabel: 'agent3',
                active: true,
                monitored: true,
                caCommErrors: 1,
            },
            {
                id: 3,
                name: 'dhcp4',
                machineId: 3,
                machineLabel: 'agent3',
                active: true,
                monitored: true,
                agentCommErrors: 0,
            },
            // Kea daemons on agent4.
            {
                id: 1,
                name: 'ca',
                machineId: 4,
                machineLabel: 'agent4',
                active: true,
                monitored: true,
                caCommErrors: 1,
            },
            {
                id: 3,
                name: 'dhcp4',
                machineId: 4,
                machineLabel: 'agent4',
                active: true,
                monitored: true,
                agentCommErrors: 5,
                daemonCommErrors: 4,
            },
            // Bind9 daemon on agent5.
            {
                id: 5,
                name: 'named',
                machineId: 5,
                machineLabel: 'agent5',
                active: true,
                monitored: true,
                agentCommErrors: 5,
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
                id: 6,
                machineId: 6,
                machineLabel: 'agent6',
                name: 'named',
                daemonCommErrors: 6,
                monitored: true,
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
                id: 7,
                machineId: 1,
                machineLabel: 'agent1',
                name: 'named',
                statsCommErrors: 7,
                monitored: true,
            },
        ]
        component.ngOnInit()
        expect(component.nodes.length).toBe(6)

        // Stork agent on agent1.
        expect(component.nodes[0].icon).toBe('pi pi-server')
        expect(component.nodes[0].type).toBe('machine')
        expect(component.nodes[0].children.length).toBe(4)
        expect(component.nodes[0].styleClass).toBe('communication-failing')
        expect(component.nodes[0].expanded).toBeTrue()
        expect(component.nodes[0].data?.attrs?.machineId).toBe(1)
        expect(component.nodes[0].data?.attrs?.machineLabel).toBe('agent1')

        // Kea agent on agent1.
        expect(component.nodes[0].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[0].children[0].type).toBe('kea')
        expect(component.nodes[0].children[0].children).toBeUndefined()
        expect(component.nodes[0].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[0].expanded).toBeTrue()
        expect(component.nodes[0].children[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[0].children[0].data?.attrs?.name).toBe('ca')

        // DHCPv4 server on agent1.
        expect(component.nodes[0].children[1].icon).toBe('pi pi-sitemap')
        expect(component.nodes[0].children[1].type).toBe('kea')
        expect(component.nodes[0].children[1].children).toBeFalsy()
        expect(component.nodes[0].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[1].data?.attrs?.id).toBe(3)
        expect(component.nodes[0].children[1].data?.attrs?.name).toBe('dhcp4')

        // named control channel on agent1.
        expect(component.nodes[0].children[2].icon).toBe('pi pi-link')
        expect(component.nodes[0].children[2].type).toBe('named-channel')
        expect(component.nodes[0].children[2].children).toBeFalsy()
        expect(component.nodes[0].children[2].styleClass).toBe('communication-ok')
        expect(component.nodes[0].children[2].data?.attrs?.id).toBe(7)
        expect(component.nodes[0].children[2].data?.attrs?.name).toBe('named')
        expect(component.nodes[0].children[2].data?.channelName).toBe('Control')

        // named stats channel on agent1.
        expect(component.nodes[0].children[3].icon).toBe('pi pi-link')
        expect(component.nodes[0].children[3].type).toBe('named-channel')
        expect(component.nodes[0].children[3].children).toBeFalsy()
        expect(component.nodes[0].children[3].styleClass).toBe('communication-failing')
        expect(component.nodes[0].children[3].data?.attrs?.id).toBe(7)
        expect(component.nodes[0].children[3].data?.attrs?.name).toBe('named')
        expect(component.nodes[0].children[3].data?.channelName).toBe('Statistics')

        // Stork agent on agent2.
        expect(component.nodes[1].icon).toBe('pi pi-server')
        expect(component.nodes[1].type).toBe('machine')
        expect(component.nodes[1].children.length).toBe(4)
        expect(component.nodes[1].styleClass).toBe('communication-ok')
        expect(component.nodes[1].expanded).toBeTrue()
        expect(component.nodes[1].data?.attrs?.machineId).toBe(2)
        expect(component.nodes[1].data?.attrs?.machineLabel).toBe('agent2')

        // Kea agent on agent2.
        expect(component.nodes[1].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[1].children[0].type).toBe('kea')
        expect(component.nodes[1].children[0].children).toBeUndefined()
        expect(component.nodes[1].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[1].children[0].expanded).toBeTrue()
        expect(component.nodes[1].children[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[1].children[0].data?.attrs?.name).toBe('ca')

        // DDNS server on agent2.
        expect(component.nodes[1].children[1].icon).toBe('pi pi-sitemap')
        expect(component.nodes[1].children[1].type).toBe('kea')
        expect(component.nodes[1].children[1].children).toBeFalsy()
        expect(component.nodes[1].children[1].styleClass).toBe('communication-disabled')
        expect(component.nodes[1].children[1].data?.attrs?.id).toBe(2)
        expect(component.nodes[1].children[1].data?.attrs?.name).toBe('d2')

        // DHCPv4 server on agent2.
        expect(component.nodes[1].children[2].icon).toBe('pi pi-sitemap')
        expect(component.nodes[1].children[2].type).toBe('kea')
        expect(component.nodes[1].children[2].children).toBeFalsy()
        expect(component.nodes[1].children[2].styleClass).toBe('communication-ok')
        expect(component.nodes[1].children[2].data?.attrs?.id).toBe(3)
        expect(component.nodes[1].children[2].data?.attrs?.name).toBe('dhcp4')

        // DHCPv6 server on agent2.
        expect(component.nodes[1].children[3].icon).toBe('pi pi-sitemap')
        expect(component.nodes[1].children[3].type).toBe('kea')
        expect(component.nodes[1].children[3].children).toBeFalsy()
        expect(component.nodes[1].children[3].styleClass).toBe('communication-disabled')
        expect(component.nodes[1].children[3].data?.attrs?.id).toBe(4)
        expect(component.nodes[1].children[3].data?.attrs?.name).toBe('dhcp6')

        // Stork agent on agent3.
        expect(component.nodes[2].icon).toBe('pi pi-server')
        expect(component.nodes[2].type).toBe('machine')
        expect(component.nodes[2].children.length).toBe(2)
        expect(component.nodes[2].styleClass).toBe('communication-ok')
        expect(component.nodes[2].expanded).toBeTrue()
        expect(component.nodes[2].data?.attrs?.machineId).toBe(3)
        expect(component.nodes[2].data?.attrs?.machineLabel).toBe('agent3')

        // Kea agent on agent3.
        expect(component.nodes[2].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[2].children[0].type).toBe('kea')
        expect(component.nodes[2].children[0].children).toBeUndefined()
        expect(component.nodes[2].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[2].children[0].expanded).toBeTrue()
        expect(component.nodes[2].children[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[2].children[0].data?.attrs?.name).toBe('ca')

        // Kea DHCPv4 on agent3.
        expect(component.nodes[2].children[1].icon).toBe('pi pi-sitemap')
        expect(component.nodes[2].children[1].type).toBe('kea')
        expect(component.nodes[2].children[1].children).toBeUndefined()
        expect(component.nodes[2].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[2].children[1].expanded).toBeTrue()
        expect(component.nodes[2].children[1].data?.attrs?.id).toBe(3)
        expect(component.nodes[2].children[1].data?.attrs?.name).toBe('dhcp4')

        // Stork agent on agent4.
        expect(component.nodes[3].icon).toBe('pi pi-server')
        expect(component.nodes[3].type).toBe('machine')
        expect(component.nodes[3].children.length).toBe(2)
        expect(component.nodes[3].styleClass).toBe('communication-failing')
        expect(component.nodes[3].expanded).toBeTrue()
        expect(component.nodes[3].data?.attrs?.machineId).toBe(4)
        expect(component.nodes[3].data?.attrs?.machineLabel).toBe('agent4')

        // Kea agent on agent4.
        expect(component.nodes[3].children[0].icon).toBe('pi pi-sitemap')
        expect(component.nodes[3].children[0].type).toBe('kea')
        expect(component.nodes[3].children[0].children).toBeUndefined()
        expect(component.nodes[3].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[3].children[0].expanded).toBeTrue()
        expect(component.nodes[3].children[0].data?.attrs?.id).toBe(1)
        expect(component.nodes[3].children[0].data?.attrs?.name).toBe('ca')

        // DHCPv4 server on agent4.
        expect(component.nodes[3].children[1].icon).toBe('pi pi-sitemap')
        expect(component.nodes[3].children[1].type).toBe('kea')
        expect(component.nodes[3].children[1].children).toBeFalsy()
        expect(component.nodes[3].children[1].styleClass).toBe('communication-failing')
        expect(component.nodes[3].children[1].data?.attrs?.id).toBe(3)
        expect(component.nodes[3].children[1].data?.attrs?.name).toBe('dhcp4')

        // Stork agent on agent5.
        expect(component.nodes[4].icon).toBe('pi pi-server')
        expect(component.nodes[4].type).toBe('machine')
        expect(component.nodes[4].children.length).toBe(2)
        expect(component.nodes[4].styleClass).toBe('communication-failing')
        expect(component.nodes[4].expanded).toBeTrue()
        expect(component.nodes[4].data?.attrs?.machineId).toBe(5)
        expect(component.nodes[4].data?.attrs?.machineLabel).toBe('agent5')

        // named control channel on agent5.
        expect(component.nodes[4].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[4].children[0].type).toBe('named-channel')
        expect(component.nodes[4].children[0].children).toBeFalsy()
        expect(component.nodes[4].children[0].styleClass).toBe('communication-ok')
        expect(component.nodes[4].children[0].data?.attrs?.id).toBe(5)
        expect(component.nodes[4].children[0].data?.attrs?.name).toBe('named')
        expect(component.nodes[4].children[0].data?.channelName).toBe('Control')

        // named statistics channel on agent5.
        expect(component.nodes[4].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[4].children[1].type).toBe('named-channel')
        expect(component.nodes[4].children[1].children).toBeFalsy()
        expect(component.nodes[4].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[4].children[1].data?.attrs?.id).toBe(5)
        expect(component.nodes[4].children[1].data?.attrs?.name).toBe('named')
        expect(component.nodes[4].children[1].data?.channelName).toBe('Statistics')

        // Stork agent on agent6.
        expect(component.nodes[5].icon).toBe('pi pi-server')
        expect(component.nodes[5].type).toBe('machine')
        expect(component.nodes[5].children.length).toBe(2)
        expect(component.nodes[5].styleClass).toBe('communication-ok')
        expect(component.nodes[5].expanded).toBeTrue()
        expect(component.nodes[5].data?.attrs?.machineId).toBe(6)
        expect(component.nodes[5].data?.attrs?.machineLabel).toBe('agent6')

        // named control channel on agent6.
        expect(component.nodes[5].children[0].icon).toBe('pi pi-link')
        expect(component.nodes[5].children[0].type).toBe('named-channel')
        expect(component.nodes[5].children[0].children).toBeFalsy()
        expect(component.nodes[5].children[0].styleClass).toBe('communication-failing')
        expect(component.nodes[5].children[0].data?.attrs?.id).toBe(6)
        expect(component.nodes[5].children[0].data?.attrs?.name).toBe('named')
        expect(component.nodes[5].children[0].data?.channelName).toBe('Control')

        // named statistics channel on agent6.
        expect(component.nodes[5].children[1].icon).toBe('pi pi-link')
        expect(component.nodes[5].children[1].type).toBe('named-channel')
        expect(component.nodes[5].children[1].children).toBeFalsy()
        expect(component.nodes[5].children[1].styleClass).toBe('communication-ok')
        expect(component.nodes[5].children[1].data?.attrs?.id).toBe(6)
        expect(component.nodes[5].children[1].data?.attrs?.name).toBe('named')
        expect(component.nodes[5].children[1].data?.channelName).toBe('Statistics')
    })
})
