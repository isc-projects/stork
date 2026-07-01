import { ZoneTransfersPageComponent } from './zone-transfers-page.component'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideRouter, withHashLocation } from '@angular/router'
import { toastDecorator } from '../utils-stories'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ZoneTransferState } from '../backend'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { datetimeToLocal } from '../utils'

const meta: Meta<ZoneTransfersPageComponent> = {
    title: 'App/ZoneTransfersPage',
    component: ZoneTransfersPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideRouter(
                    [
                        {
                            path: 'dns/zone-transfer-states',
                            component: ZoneTransfersPageComponent,
                        },
                        {
                            path: '**',
                            component: ZoneTransfersPageComponent,
                        },
                    ],
                    withHashLocation()
                ),
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
            ],
        }),
        moduleMetadata({
            providers: [ConfirmationService],
        }),
        toastDecorator,
    ],
    async beforeEach() {
        localStorage.removeItem('zone-transfers-table-filters-toolbar-shown')
    },
}

const allZoneTransfers: ZoneTransferState[] = [
    {
        id: 1,
        createdAt: '2026-01-01T10:41:12Z',
        viewName: '_default',
        zoneName: 'zone.example.org',
        client: '192.0.2.1',
        server: '192.0.2.2',
        serial: 1234567890,
        status: 'completed',
        startedAt: '2026-01-01T10:40:27Z',
        completedAt: '2026-01-01T10:45:11Z',
        duration: '2h3m10.451s',
        bytesPerSecond: 12,
        message:
            'AXFR completed: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)',
        messagesCount: 79,
        recordsCount: 24872,
        bytesCount: 1320233,
    },
    {
        id: 2,
        createdAt: '2026-01-01T10:48:01Z',
        viewName: '_default',
        zoneName: 'zone2.example.org',
        client: '2001:db8::1',
        server: '2001:db8::2',
        serial: 234567891,
        status: 'completed',
        startedAt: '2026-01-01T10:47:30Z',
        completedAt: '2026-01-01T10:49:30Z',
        duration: '2h3m10.451s',
        bytesPerSecond: 122304,
        clientMachineID: 1,
        clientMachineAddress: 'agent-bind9',
        serverMachineID: 2,
        serverMachineAddress: 'agent-bind9-2',
        message:
            'AXFR completed: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)',
        messagesCount: 79,
        recordsCount: 24872,
        bytesCount: 1320233,
    },
    {
        id: 3,
        createdAt: '2026-01-01T10:49:41Z',
        viewName: 'public',
        zoneName: 'zone3.example.org',
        client: '2001:db8::1',
        server: '2001:db8::2',
        serial: 3456789012,
        status: 'started',
        startedAt: '2026-01-01T10:49:42Z',
        duration: '0.351s',
        bytesPerSecond: 20234434234,
        message: 'AXFR started',
        messagesCount: 0,
        recordsCount: 0,
        bytesCount: 0,
        serverMachineID: 2,
        serverMachineAddress: 'agent-bind9-2',
    },
    {
        id: 4,
        createdAt: '2026-01-01T10:55:22Z',
        viewName: 'public',
        zoneName: '.',
        client: '::1',
        server: '2001:db8::2',
        serial: 4567890123,
        status: 'message',
        startedAt: '2026-01-01T10:49:39Z',
        message: 'AXFR failed with timeout',
        messagesCount: 0,
        recordsCount: 0,
        bytesCount: 0,
        serverMachineID: 1,
        serverMachineAddress: 'agent-bind9',
        local: true,
    },
]

/**
 * Retrieves query parameters from the provided URL and filters the zone transfers accordingly.
 *
 * @param url - The URL to retrieve the query parameters from.
 * @returns The filtered zone transfers.
 */
function getFilteredZoneTransfers(url: any): ZoneTransferState[] {
    const search = new URL(url, 'http://localhost').search
    const searchParams = new URLSearchParams(search)

    // Sort the zone transfers by the specified field.
    let sortField = searchParams.get('sortField')
    switch (sortField) {
        case 'bytes_per_second':
            sortField = 'bytesPerSecond'
            break
        case 'started_at':
            sortField = 'startedAt'
            break
        case 'completed_at':
            sortField = 'completedAt'
            break
    }
    const sortDir = searchParams.get('sortDir')
    let filteredZoneTransfers = allZoneTransfers.sort((first, second) => {
        if (sortField === 'bytesPerSecond') {
            // Sorting by rate requires special handling because it is a number
            // and can be null.
            const firstBps = first.bytesPerSecond ?? 0
            const secondBps = second.bytesPerSecond ?? 0
            return sortDir === 'asc' ? firstBps - secondBps : secondBps - firstBps
        }
        // Remaining sort fields are strings.
        return sortDir === 'asc'
            ? (first[sortField] ?? '').localeCompare(second[sortField] ?? '')
            : (second[sortField] ?? '').localeCompare(first[sortField] ?? '')
    })
    // Filter by status.
    if (searchParams.has('status')) {
        filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
            return searchParams.getAll('status').includes(zoneTransfer.status)
        })
    }
    // Filter by partial serial.
    if (searchParams.has('serial')) {
        filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
            return zoneTransfer.serial?.toString().includes(searchParams.get('serial') ?? '')
        })
    }
    // filter by primary.
    if (searchParams.has('serverMachineId')) {
        filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
            const machineIdParams = searchParams.getAll('serverMachineId')
            return machineIdParams.includes(zoneTransfer.serverMachineID?.toString())
        })
    }
    // Filter by secondary.
    if (searchParams.has('clientMachineId')) {
        filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
            const machineIdParams = searchParams.getAll('clientMachineId')
            return machineIdParams.includes(zoneTransfer.clientMachineID?.toString())
        })
    }
    // Filter by partial zone name, client, server or message.
    if (searchParams.has('text')) {
        filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
            for (const field of ['zoneName', 'client', 'server', 'message']) {
                if (zoneTransfer[field]?.includes(searchParams.get('text') ?? '')) {
                    return true
                }
            }
            return false
        })
    }
    // Return local transfers if checked.
    let includeLocal = searchParams.get('includeLocal') === 'true'
    filteredZoneTransfers = filteredZoneTransfers.filter((zoneTransfer) => {
        return !zoneTransfer.local || zoneTransfer.local === includeLocal
    })
    return filteredZoneTransfers
}

export default meta
type Story = StoryObj<ZoneTransfersPageComponent>

const zoneTransferMockDataUrls = [
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&status=started',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&status=started&status=connected',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&status=started&status=connected&status=completed',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&status=started&status=connected&status=completed&status=message',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&serial=:serial',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&text=:text',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&sortField=:sortField&sortDir=:sortDir',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&status=connected&sortField=:sortField&sortDir=:sortDir',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&serverMachineId=:serverMachineId',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&clientMachineId=:clientMachineId',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&serverMachineId=:serverMachineId&clientMachineId=:clientMachineId',
    'http://localhost/api/zone-transfer-states?start=:start&limit=:limit&includeLocal=:includeLocal',
]

/**
 * Creates mock data for the zone transfers API.
 *
 * @param urlPattern URL pattern to use for the mock data.
 * @returns Mock data for the zone transfers API.
 */
function createZoneTransferMockData(urlPattern: string) {
    return {
        url: urlPattern,
        method: 'GET',
        status: 200,
        response: ({ url }) => {
            const filteredZoneTransfers = getFilteredZoneTransfers(url)
            return {
                items: filteredZoneTransfers,
                total: filteredZoneTransfers.length,
            }
        },
    }
}

export const ListZoneTransfers: Story = {
    parameters: {
        mockData: [
            ...zoneTransferMockDataUrls.map(createZoneTransferMockData),
            {
                url: 'http://localhost/api/machines/directory?daemonName=named',
                method: 'GET',
                delay: 0,
                status: 200,
                response: () => {
                    return {
                        items: [
                            { id: 1, address: 'agent-bind9' },
                            { id: 2, address: 'agent-bind9-2' },
                        ],
                        total: 2,
                    }
                },
            },
        ],
    },
}

export const TestListAllZoneTransfers: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListZoneTransfers.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const table = await canvas.findByRole('table')
        const body = within(canvasElement.parentElement)

        // Find and click the checkbox that includes local transfers.
        let includeLocalCheckbox = canvas.getByLabelText('Include local transfers')
        await expect(includeLocalCheckbox).toBeTruthy()
        await userEvent.click(includeLocalCheckbox)
        // Wait for all transfers to be displayed including the local ones.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(allZoneTransfers.length + 1))

        // Find all rows in the table.
        const rows = await within(table).findAllByRole('row')

        // Make sure that the table headers are displayed.
        await expect(within(rows[0]).getByText('Started At')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Zone')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('View')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Serial')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Duration')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Rate')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Status')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Primary')).toBeInTheDocument()
        await expect(within(rows[0]).getByText('Secondary')).toBeInTheDocument()

        // Validate that the table rows are displayed correctly.
        for (let i = 1; i < rows.length; i++) {
            const row = rows[i]
            const zoneTransfer = allZoneTransfers[i - 1]

            // First column should display the local transfer local start time.
            await expect(within(row).getByText(datetimeToLocal(zoneTransfer.startedAt))).toBeInTheDocument()
            if (zoneTransfer.zoneName !== '.') {
                // Non-root zone name is displayed as is.
                await expect(within(row).getByText(zoneTransfer.zoneName)).toBeInTheDocument()
            } else {
                // Root zone is displayed as (root).
                await expect(within(row).getByText('(root)')).toBeInTheDocument()
            }
            // view name
            await expect(within(row).getByText(zoneTransfer.viewName)).toBeInTheDocument()
            // serial
            await expect(within(row).getByText(zoneTransfer.serial.toString())).toBeInTheDocument()
            // status
            await expect(within(row).getByText(zoneTransfer.status)).toBeInTheDocument()
        }

        // The values in the remaining columns must be validated individually because
        // they are specially formatted.

        await expect(within(rows[1]).getByText('2 hours 3 minutes 10.451 seconds')).toBeInTheDocument()
        await expect(within(rows[1]).getByText('12B/s')).toBeInTheDocument()
        await expect(within(rows[1]).getByText('192.0.2.2')).toBeInTheDocument()
        await expect(within(rows[1]).getByText('192.0.2.1')).toBeInTheDocument()

        await expect(within(rows[2]).getByText('2 hours 3 minutes 10.451 seconds')).toBeInTheDocument()
        await expect(within(rows[2]).getByText('122.3kB/s')).toBeInTheDocument()
        await expect(within(rows[2]).getByText('[2] agent-bind9-2')).toBeInTheDocument()
        await expect(within(rows[2]).getByText('[1] agent-bind9')).toBeInTheDocument()

        await expect(within(rows[3]).getByText('0.351 seconds')).toBeInTheDocument()
        await expect(within(rows[3]).getByText('20.2GB/s')).toBeInTheDocument()
        await expect(within(rows[3]).getByText('[2] agent-bind9-2')).toBeInTheDocument()
        await expect(within(rows[3]).getByText('2001:db8::1')).toBeInTheDocument()

        await expect(within(rows[4]).getAllByText('N/A')).toHaveLength(2)
        await expect(within(rows[4]).getByText('[1] agent-bind9')).toBeInTheDocument()
        await expect(within(rows[4]).getByText('::1')).toBeInTheDocument()
        await expect(within(rows[4]).getByText('local')).toBeInTheDocument()

        // Validate that hovering over the primary machine displays the primary address.
        await userEvent.hover(within(rows[2]).getByText('[2] agent-bind9-2'))
        await expect(body.getByRole('tooltip', { name: 'Primary Address: 2001:db8::2' })).toBeInTheDocument()
        await userEvent.unhover(within(rows[2]).getByText('[2] agent-bind9-2'))

        // Validate that hovering over the secondary machine displays the secondary address.
        await userEvent.hover(within(rows[2]).getByText('[1] agent-bind9'))
        await expect(body.getByRole('tooltip', { name: 'Secondary Address: 2001:db8::1' })).toBeInTheDocument()
        await userEvent.unhover(within(rows[2]).getByText('[1] agent-bind9'))

        // Make sure that the selected row can be expanded.
        const expandBtn = await within(rows[2]).findByRole('button')
        await userEvent.click(expandBtn)

        // Validate that the expanded row displays the correct information.
        await expect(within(table).getByText(datetimeToLocal(allZoneTransfers[1].createdAt))).toBeInTheDocument()
        await expect(within(table).getByText(datetimeToLocal(allZoneTransfers[1].completedAt))).toBeInTheDocument()
        await expect(within(table).getByText(allZoneTransfers[1].messagesCount)).toBeInTheDocument()
        await expect(within(table).getByText(allZoneTransfers[1].recordsCount)).toBeInTheDocument()
        await expect(within(table).getByText(allZoneTransfers[1].bytesCount)).toBeInTheDocument()
        await expect(within(table).getByText(allZoneTransfers[1].message)).toBeInTheDocument()
    },
}

export const TestZoneTransfersFilters: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListZoneTransfers.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        // Configure delay between consecutive user events to be more human-like and to give more time
        // for PrimeNG animations when automatically testing.
        const user = userEvent.setup({ delay: 50 })
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        const table = await canvas.findByRole('table')
        const comboboxes = canvas.getAllByRole('combobox') // PrimeNG p-select component has combobox role.

        await user.click(clearFiltersBtn)

        // Check filtering by zone transfer status.
        let elementID = canvas.getAllByText('Status')[0].getAttribute('for')
        let inputElement = comboboxes.find((el) => el.getAttribute('id') == elementID)
        await expect(inputElement).toBeTruthy()
        await user.click(inputElement)
        await user.keyboard('s')
        let option = await canvas.findByRole('option', { name: 'started' })
        await user.click(option)
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(2))

        // There should be a header row and the single data row with status started.
        let rows = await within(table).findAllByRole('row')
        await expect(rows).toHaveLength(2)
        await expect(within(rows[1]).getByText('started')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)

        // Check filtering by serial.
        inputElement = canvas.getByLabelText('Serial')
        await expect(inputElement).toBeTruthy()
        await user.click(inputElement)
        // Type partial serial number in  the input field.
        await user.type(inputElement, '123')
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(2))

        // There should be a header row and the single data row with serial 1234567890.
        rows = await within(table).findAllByRole('row')
        await expect(rows).toHaveLength(2)
        await expect(within(rows[1]).getByText('1234567890')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)

        // Check filtering by primary machine.
        elementID = canvas.getAllByText('Primary')[0].getAttribute('for')
        inputElement = comboboxes.find((el) => el.getAttribute('id') == elementID)
        await expect(inputElement).toBeTruthy()
        await user.click(inputElement)
        await user.keyboard('a')
        option = await canvas.findByRole('option', { name: 'agent-bind9-2' })
        await user.click(option)
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(3))

        rows = await within(table).findAllByRole('row')
        await expect(within(rows[1]).getByText('zone2.example.org')).toBeInTheDocument()
        await expect(within(rows[2]).getByText('zone3.example.org')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)

        // Check filtering by secondary machine.
        elementID = canvas.getAllByText('Secondary')[0].getAttribute('for')
        inputElement = comboboxes.find((el) => el.getAttribute('id') == elementID)
        await expect(inputElement).toBeTruthy()
        await user.click(inputElement)
        await user.keyboard('a')

        option = await canvas.findByRole('option', { name: 'agent-bind9' })
        await user.click(option)
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(2))

        rows = await within(table).findAllByRole('row')
        await expect(rows).toHaveLength(2)
        await expect(within(rows[1]).getByText('zone2.example.org')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)

        // Check filtering by partial zone name.
        inputElement = canvas.getByPlaceholderText('Search')
        await expect(inputElement).toBeTruthy()
        await user.type(inputElement, 'zone3.')
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(2))

        rows = await within(table).findAllByRole('row')
        await expect(rows).toHaveLength(2)
        await expect(within(rows[1]).getByText('zone3.example.org')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)

        // Test including the local transfers.
        inputElement = canvas.getByLabelText('Include local transfers')
        await expect(inputElement).toBeTruthy()
        await user.click(inputElement)
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(5))

        // Make sure that all transfers are displayed including the local ones.
        rows = await within(table).findAllByRole('row')
        await expect(rows).toHaveLength(5)
        await expect(within(rows[4]).getByText('(root)')).toBeInTheDocument()

        // Clear the filter.
        await user.click(clearFiltersBtn)
    },
}
