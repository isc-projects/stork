import { DaemonFilterComponent } from './daemon-filter.component'
import { applicationConfig, argsToTemplate, Meta, StoryObj } from '@storybook/angular'
import { provideHttpClient } from '@angular/common/http'
import { userEvent, within, expect, waitFor } from '@storybook/test'

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
        label: 'pdns_server@agent-pdns',
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
        label: 'named@agent-bind9',
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
        label: 'named@agent-bind9-2',
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
        label: 'CA@agent-kea6',
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
        label: 'DHCPv6@agent-kea6',
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
        label: 'CA@agent-kea',
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
        label: 'DHCPv4@agent-kea',
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
        label: 'CA@agent-kea-ha1',
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
        label: 'DHCPv4@agent-kea-ha1',
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
        label: 'CA@agent-kea-ha2',
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
        label: 'DHCPv4@agent-kea-ha2',
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
        label: 'DHCPv6@agent-kea-ha2',
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
        label: 'CA@agent-kea-ha3',
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
        label: 'DHCPv4@agent-kea-ha3',
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
        label: 'DHCPv6@agent-kea-ha3',
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
        daemonNames: {
            control: { type: 'multi-select' },
            options: ['dhcp4', 'dhcp6', 'named', 'pdns', 'ca', 'd2', 'netconf'],
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
        ],
    },
    render: (args) => ({
        props: args,
        template: `
            <app-daemon-filter (daemonIDChange)="output.value=$event" (errorOccurred)="err.value=$event" ${argsToTemplate(args)}></app-daemon-filter>
            <hr />
            Selected daemon ID:
            <input #output placeholder="daemonID" disabled />
            <button type="button">Dummy button</button>
            <hr />
            Error emitted:
            <input #err placeholder="error" disabled class="w-full" />
            `,
    }),
} as Meta

type Story = StoryObj<DaemonFilterComponent>

export const AllDaemons: Story = {}

export const SlowBackendResponses: Story = {
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
                delay: 2000,
            },
        ],
    },
}

export const TimeoutOnBackendResponse: Story = {
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
                delay: 4000,
            },
        ],
    },
}

export const PreselectedDaemon: Story = {
    args: {
        daemonID: 57,
    },
}

export const DHCPDemons: Story = {
    args: {
        daemonNames: ['dhcp4', 'dhcp6'],
    },
}

export const DNSDemons: Story = {
    args: {
        daemonNames: ['named', 'pdns'],
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
        ],
    },
}

export const TestNormalUsage: Story = {
    args: {
        daemonNames: ['dhcp4', 'dhcp6', 'named', 'pdns'],
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const user = userEvent.setup({ delay: 100 })

        // Act + Assert
        const input = await canvas.findByRole('combobox')

        // Test daemon lookup.
        await user.click(input)
        await user.keyboard('name')

        // Two daemons are expected.
        const options = await canvas.findAllByRole('option')
        await expect(options).toHaveLength(2)

        // Pick an option.
        await user.click(options[0])
        const daemonID = await canvas.findByPlaceholderText('daemonID')
        await expect(daemonID).toHaveValue('57')

        await user.clear(input)
        const dummyButton = await canvas.findByRole('button', { name: 'Dummy button' })
        await user.click(dummyButton) // in order to lose focus on autocomplete
        await expect(daemonID).toHaveValue('')

        // Use the autocomplete dropdown with keyboard.
        await user.click(input)
        await user.keyboard('{Tab}{Enter}')
        const listbox = await canvas.findByRole('listbox')

        // All daemons should be displayed.
        const allOptions = await within(listbox).findAllByRole('option')
        const acceptedDaemons = allDaemons.filter((d) => ['dhcp4', 'dhcp6', 'named', 'pdns'].includes(d.name))
        await waitFor(() => expect(allOptions).toHaveLength(acceptedDaemons.length))

        // Pick an option.
        await user.keyboard('{ArrowDown}{ArrowDown}{ArrowDown}{ArrowDown}{Enter}')
        await waitFor(() => expect(daemonID).toHaveValue('60'))
    },
}

export const TestSlowBackendResponse: Story = {
    parameters: SlowBackendResponses.parameters,
    args: {
        daemonNames: ['dhcp4', 'dhcp6', 'named', 'pdns'],
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const user = userEvent.setup({ delay: 100 })

        // Act + Assert
        const input = await canvas.findByRole('combobox')

        // Test daemon lookup.
        await user.click(input)
        await user.keyboard('name')

        // Two daemons are expected. We have to wait, but no longer than timeout.
        await waitFor(() => expect(canvas.getAllByRole('option')).toHaveLength(2), { timeout: 2500 })
        const options = await canvas.findAllByRole('option')
        await expect(options).toHaveLength(2)

        // Pick an option.
        await user.click(options[0])
        const daemonID = await canvas.findByPlaceholderText('daemonID')
        await expect(daemonID).toHaveValue('57')

        await user.clear(input)
        const dummyButton = await canvas.findByRole('button', { name: 'Dummy button' })
        await user.click(dummyButton) // in order to lose focus on autocomplete
        await expect(daemonID).toHaveValue('')

        // Use the autocomplete dropdown with keyboard. Once the daemons directory was fetched from backend, lookups are fast.
        await user.click(input)
        await user.keyboard('{Tab}{Enter}')
        const listbox = await canvas.findByRole('listbox')

        // All daemons should be displayed.
        const allOptions = await within(listbox).findAllByRole('option')
        const acceptedDaemons = allDaemons.filter((d) => ['dhcp4', 'dhcp6', 'named', 'pdns'].includes(d.name))
        await waitFor(() => expect(allOptions).toHaveLength(acceptedDaemons.length))

        // Pick an option.
        await user.keyboard('{ArrowDown}{ArrowDown}{ArrowDown}{ArrowDown}{Enter}')
        await waitFor(() => expect(daemonID).toHaveValue('60'))
    },
}

export const TestTimeoutResponse: Story = {
    tags: ['no-test-in-ci'], // Skip in CI because it needs more than 2500ms to run.
    parameters: TimeoutOnBackendResponse.parameters,
    args: {
        daemonNames: ['dhcp4', 'dhcp6', 'named', 'pdns'],
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)

        // Act + Assert
        const input = await canvas.findByRole('combobox')
        const errorInput = await canvas.findByPlaceholderText('error')
        const daemonID = await canvas.findByPlaceholderText('daemonID')

        // Test daemon lookup.
        await userEvent.click(input)
        await userEvent.keyboard('name')

        // Timeout error is expected.
        await waitFor(
            () =>
                expect(errorInput).toHaveValue(
                    'Failed to retrieve daemons from Stork server: timeout - no response in 2500ms'
                ),
            { timeout: 2500 }
        )
        await expect(daemonID).toHaveValue('')
    },
}
