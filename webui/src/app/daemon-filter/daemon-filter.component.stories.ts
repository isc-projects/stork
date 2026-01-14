import { DaemonFilterComponent } from './daemon-filter.component'
import { applicationConfig, argsToTemplate, Meta, StoryObj } from '@storybook/angular'
import { provideHttpClient } from '@angular/common/http'

const allDaemons = [
    {
        active: true,
        id: 56,
        machine: {
            address: 'agent-pdns',
            agentPort: 8891,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-pdns',
            id: 29,
        },
        name: 'pdns',
        version: '4.7.3',
    },
    {
        active: true,
        id: 57,
        machine: {
            address: 'agent-bind9',
            agentPort: 8883,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-bind9',
            id: 31,
        },
        name: 'named',
        version: 'BIND 9.20.16 (Stable Release) <id:c97aa2d>',
    },
    {
        active: true,
        id: 58,
        machine: {
            address: 'agent-bind9-2',
            agentPort: 8882,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-bind9-2',
            id: 32,
        },
        name: 'named',
        version: 'BIND 9.20.16 (Stable Release) <id:c97aa2d>',
    },
    {
        active: true,
        id: 59,
        machine: {
            address: 'agent-kea6',
            agentPort: 8887,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea6',
            id: 34,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 60,
        machine: {
            address: 'agent-kea6',
            agentPort: 8887,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea6',
            id: 34,
        },
        name: 'dhcp6',
        version: '3.1.0',
    },
    {
        active: true,
        id: 61,
        machine: {
            address: 'agent-kea',
            agentPort: 8888,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea',
            id: 35,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 62,
        machine: {
            address: 'agent-kea',
            agentPort: 8888,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea',
            id: 35,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 63,
        machine: {
            address: 'agent-kea-ha1',
            agentPort: 8886,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha1',
            id: 36,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 64,
        machine: {
            address: 'agent-kea-ha1',
            agentPort: 8886,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha1',
            id: 36,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 65,
        machine: {
            address: 'agent-kea-ha2',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2',
            id: 37,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 66,
        machine: {
            address: 'agent-kea-ha2',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2',
            id: 37,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 67,
        machine: {
            address: 'agent-kea-ha2',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2',
            id: 37,
        },
        name: 'dhcp6',
        version: '3.1.0',
    },
    {
        active: true,
        id: 68,
        machine: {
            address: 'agent-kea-ha3',
            agentPort: 8890,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha3',
            id: 38,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 69,
        machine: {
            address: 'agent-kea-ha3',
            agentPort: 8890,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha3',
            id: 38,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 70,
        machine: {
            address: 'agent-kea-ha3',
            agentPort: 8890,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha3',
            id: 38,
        },
        name: 'dhcp6',
        version: '3.1.0',
    },
]

export default {
    title: 'App/DaemonFilter',
    component: DaemonFilterComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient()],
        }),
    ],
    argTypes: {
        domain: {
            control: { type: 'radio' },
            options: [undefined, 'dns', 'dhcp'],
        },
    },
    parameters: {
        mockData: [
            {
                url: 'api/daemons/directory',
                method: 'GET',
                status: 200,
                response: () => ({
                    items: allDaemons,
                    total: allDaemons.length,
                }),
                delay: 500,
            },
            {
                url: 'api/daemons/directory?text=t',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const text = req.searchParams.text
                    const filtered = allDaemons.filter(
                        (d) =>
                            d.name.includes(text) ||
                            d.machine.hostname.includes(text) ||
                            d.machine.address.includes(text)
                    )
                    return { items: filtered, total: filtered.length }
                },
                delay: 500,
            },
            {
                url: 'api/daemons/directory?domain=d',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (!req.searchParams.domain) {
                        return { items: allDaemons, total: allDaemons.length }
                    }
                    const isDns = req.searchParams.domain == 'dns'
                    const inDomain = allDaemons.filter((d) => {
                        return isDns
                            ? ['named', 'pdns'].includes(d.name)
                            : ['dhcp4', 'dhcp6', 'netconf', 'd2', 'ca'].includes(d.name)
                    })
                    return { items: inDomain, total: inDomain.length }
                },
                delay: 500,
            },
            {
                url: 'api/daemons/directory?text=t&domain=d',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const text = req.searchParams.text
                    const filtered = allDaemons.filter(
                        (d) =>
                            d.name.includes(text) ||
                            d.machine.hostname.includes(text) ||
                            d.machine.address.includes(text)
                    )
                    if (!req.searchParams.domain) {
                        return { items: filtered, total: filtered.length }
                    }
                    const isDns = req.searchParams.domain == 'dns'
                    const inDomain = filtered.filter((d) => {
                        return isDns
                            ? ['named', 'pdns'].includes(d.name)
                            : ['dhcp4', 'dhcp6', 'netconf', 'd2', 'ca'].includes(d.name)
                    })
                    return { items: inDomain, total: inDomain.length }
                },
                delay: 500,
            },
        ],
    },
    render: (args) => ({
        props: args,
        template: `
            <app-daemon-filter (daemonIDChange)="output.value=$event" (errorOccurred)="err.value=$event" ${argsToTemplate(args)}></app-daemon-filter>
            <hr />
            Selected daemon ID:
            <input #output disabled />
            <hr />
            Error emitted:
            <input #err disabled />
            `,
    }),
} as Meta

type Story = StoryObj<DaemonFilterComponent>

export const AllDomains: Story = {
    args: {
        domain: undefined,
    },
}

export const PreselectedDaemon: Story = {
    args: {
        domain: undefined,
        daemonID: 57,
    },
}

export const DHCPDomain: Story = {
    args: {
        domain: 'dhcp',
    },
}

export const DNSDomain: Story = {
    args: {
        domain: 'dns',
    },
}

const daemonsMachineAddresses = [
    {
        active: true,
        id: 56,
        machine: {
            address: '3001:db8:1::cafe',
            agentPort: 8891,
            agentVersion: '2.3.2',
            id: 29,
        },
        name: 'pdns',
        version: '4.7.3',
    },
    {
        active: true,
        id: 57,
        machine: {
            address: '10.0.0.222',
            agentPort: 8883,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: '',
            id: 31,
        },
        name: 'named',
        version: 'BIND 9.20.16 (Stable Release) <id:c97aa2d>',
    },
    {
        active: true,
        id: 58,
        machine: {
            address: '10.17.0.201',
            agentPort: 8882,
            agentVersion: '2.3.2',
            daemons: [],
            id: 32,
        },
        name: 'named',
        version: 'BIND 9.20.16 (Stable Release) <id:c97aa2d>',
    },
    {
        active: true,
        id: 59,
        machine: {
            address: '3001:db8:1::face',
            agentPort: 8887,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: '',
            id: 34,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 60,
        machine: {
            address: '3001:db8:1::face',
            agentPort: 8887,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: '',
            id: 34,
        },
        name: 'dhcp6',
        version: '3.1.0',
    },
    {
        active: true,
        id: 61,
        machine: {
            address: '10.177.134.203',
            agentPort: 8888,
            agentVersion: '2.3.2',
            id: 35,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 62,
        machine: {
            address: '10.177.134.203',
            agentPort: 8888,
            agentVersion: '2.3.2',
            id: 35,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 63,
        machine: {
            address: '10.231.234.123',
            agentPort: 8886,
            agentVersion: '2.3.2',
            id: 36,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 64,
        machine: {
            address: '10.231.234.123',
            agentPort: 8886,
            agentVersion: '2.3.2',
            id: 36,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 65,
        machine: {
            address: '10.13.14.17',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2-host',
            id: 37,
        },
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 66,
        machine: {
            address: '10.13.14.17',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2-host',
            id: 37,
        },
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 67,
        machine: {
            address: '10.13.14.17',
            agentPort: 8885,
            agentVersion: '2.3.2',
            daemons: [],
            hostname: 'agent-kea-ha2-host',
            id: 37,
        },
        name: 'dhcp6',
        version: '3.1.0',
    },
    {
        active: true,
        id: 68,
        machineId: 38,
        name: 'ca',
        version: '3.1.0',
    },
    {
        active: true,
        id: 69,
        machineId: 38,
        name: 'dhcp4',
        version: '3.1.0',
    },
    {
        active: true,
        id: 70,
        machineId: 38,
        name: 'dhcp6',
        version: '3.1.0',
    },
]

export const NoHostnames: Story = {
    parameters: {
        mockData: [
            {
                url: 'api/daemons/directory',
                method: 'GET',
                status: 200,
                response: () => ({
                    items: daemonsMachineAddresses,
                    total: daemonsMachineAddresses.length,
                }),
                delay: 500,
            },
            {
                url: 'api/daemons/directory?text=t',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const text = req.searchParams.text
                    const filtered = daemonsMachineAddresses.filter(
                        (d) =>
                            d.name.includes(text) ||
                            d.machine?.hostname?.includes(text) ||
                            d.machine?.address?.includes(text)
                    )
                    return { items: filtered, total: filtered.length }
                },
                delay: 500,
            },
            {
                url: 'api/daemons/directory?domain=d',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (!req.searchParams.domain) {
                        return { items: daemonsMachineAddresses, total: daemonsMachineAddresses.length }
                    }
                    const isDns = req.searchParams.domain == 'dns'
                    const inDomain = daemonsMachineAddresses.filter((d) => {
                        return isDns
                            ? ['named', 'pdns'].includes(d.name)
                            : ['dhcp4', 'dhcp6', 'netconf', 'd2', 'ca'].includes(d.name)
                    })
                    return { items: inDomain, total: inDomain.length }
                },
                delay: 500,
            },
            {
                url: 'api/daemons/directory?text=t&domain=d',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const text = req.searchParams.text
                    const filtered = daemonsMachineAddresses.filter(
                        (d) =>
                            d.name.includes(text) ||
                            d.machine?.hostname?.includes(text) ||
                            d.machine?.address?.includes(text)
                    )
                    if (!req.searchParams.domain) {
                        return { items: filtered, total: filtered.length }
                    }
                    const isDns = req.searchParams.domain == 'dns'
                    const inDomain = filtered.filter((d) => {
                        return isDns
                            ? ['named', 'pdns'].includes(d.name)
                            : ['dhcp4', 'dhcp6', 'netconf', 'd2', 'ca'].includes(d.name)
                    })
                    return { items: inDomain, total: inDomain.length }
                },
                delay: 500,
            },
        ],
    },
}

export const ApiError: Story = {
    parameters: {
        mockData: [
            {
                url: 'api/daemons/directory',
                method: 'GET',
                status: 500,
                delay: 500,
                response: {
                    message: 'Error getting daemons directory',
                },
            },
            {
                url: 'api/daemons/directory?domain=d',
                method: 'GET',
                status: 500,
                delay: 500,
                response: {
                    message: 'Error getting daemons directory',
                },
            },
        ],
    },
}
