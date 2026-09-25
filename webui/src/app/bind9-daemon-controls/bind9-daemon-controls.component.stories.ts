import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { Bind9DaemonControlsComponent } from './bind9-daemon-controls.component'
import { toastDecorator } from '../utils-stories'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from '@openng/optimus-ui/api'
import { provideRouter } from '@angular/router'
import { expect, userEvent, waitFor, within } from 'storybook/test'

const configResponse = {
    files: [
        {
            sourcePath: '/etc/bind/named.conf',
            fileType: 'config',
            contents: ['options {', '\tlisten-on {', '\t\t127.0.0.1;', '\t};', '};'],
        },
    ],
}

const rndcKeyResponse = {
    files: [
        {
            sourcePath: '/etc/bind/rndc.key',
            fileType: 'config',
            contents: [
                'key "rndc-key" {',
                '\talgorithm hmac-sha256;',
                '\tsecret "UlJY3N2FdJ5cWUT6jQt/OPEnT9ap4b45Pzo1724yYw=";',
                '};',
            ],
        },
    ],
}

export default {
    title: 'App/Bind9DaemonControls',
    component: Bind9DaemonControlsComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi()), provideRouter([])],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?filter=config&fileSelector=config',
                method: 'GET',
                status: 200,
                delay: 500,
                response: (request) => {
                    const { fileSelector } = request.searchParams
                    if (fileSelector === 'rndc-key') {
                        return rndcKeyResponse
                    }
                    return configResponse
                },
            },
        ],
    },
} as Meta

type Story = StoryObj<Bind9DaemonControlsComponent>

export const Primary: Story = {
    args: {
        daemonId: 1,
    },
}

export const OpenConfigDialog: Story = {
    args: {
        daemonId: 1,
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: 'BIND 9 Config' }))
        const page = within(canvasElement.ownerDocument.body)
        await waitFor(() => expect(page.getByRole('dialog')).toBeInTheDocument())
    },
}

export const OpenRndcKeyDialog: Story = {
    args: {
        daemonId: 1,
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: 'RNDC Key Config' }))
        const page = within(canvasElement.ownerDocument.body)
        await waitFor(() => expect(page.getByRole('dialog')).toBeInTheDocument())
    },
}

export const ZonesButton: Story = {
    args: {
        daemonId: 1,
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const zonesButton = canvas.getByRole('button', { name: 'Zones' })
        await expect(zonesButton).toBeInTheDocument()
    },
}
