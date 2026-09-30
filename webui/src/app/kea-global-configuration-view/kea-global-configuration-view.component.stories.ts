import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from '@openng/optimus-ui/api'
import { toastDecorator } from '../utils-stories'
import { expect, fn, within } from 'storybook/test'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { DHCPOption } from '../backend'

const sampleDhcpParameters: Array<NamedCascadedParameters<object>> = [
    {
        name: 'Server1',
        parameters: [
            {
                cacheThreshold: 0.25,
                cacheMaxAge: 1000,
                clientClass: 'baz',
                controlSockets: [{ name: 'ignored' }],
                requireClientClasses: ['foo', 'bar'],
                ddnsGeneratedPrefix: 'myhost',
                ddnsOverrideClientUpdate: true,
                clientClasses: [{ name: 'ignored' }],
                optionData: [{ code: 6 }],
            },
            {
                cacheThreshold: 0.25,
                cacheMaxAge: 1000,
                clientClass: 'fbi',
                requireClientClasses: ['abc'],
                ddnsGeneratedPrefix: 'his',
                ddnsOverrideClientUpdate: false,
            },
            {
                cacheMaxAge: 1000,
                requireClientClasses: ['abc'],
                ddnsGeneratedPrefix: 'example',
                ddnsOverrideClientUpdate: true,
            },
        ],
    },
]

const sampleDhcpOptions: DHCPOption[][] = [
    [
        {
            alwaysSend: true,
            code: 5,
            fields: [
                {
                    fieldType: 'ipv4-address',
                    values: ['192.0.2.2'],
                },
            ],
            options: [],
            universe: 4,
        },
    ],
]

export default {
    title: 'App/KeaGlobalConfigurationView',
    component: KeaGlobalConfigurationViewComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), MessageService],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<KeaGlobalConfigurationViewComponent>

export const KeaGlobalConfiguration: Story = {
    args: {
        dhcpParameters: sampleDhcpParameters,
        dhcpOptions: sampleDhcpOptions,
        editBegin: fn(),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)

        const editButton = await canvas.findByRole('button', { name: 'Edit' })
        await expect(editButton).toBeEnabled()

        await expect(canvas.getByRole('group', { name: 'Global DHCP Parameters' })).toBeVisible()
        await expect(canvas.getByRole('group', { name: 'Global DHCP Options' })).toBeVisible()

        await expect(canvas.getByText('Server1')).toBeVisible()
        await expect(canvas.getByText('Cache Threshold')).toBeVisible()
        await expect(canvas.getByText('0.25')).toBeVisible()
        await expect(canvas.getByText('Cache Max Age')).toBeVisible()
        await expect(canvas.getByText('1000')).toBeVisible()
        await expect(canvas.getByText('Client Class')).toBeVisible()
        await expect(canvas.getByText('baz')).toBeVisible()
        await expect(canvas.getByText('Require Client Classes')).toBeVisible()
        await expect(canvas.getByText('foo')).toBeVisible()
        await expect(canvas.getByText('bar')).toBeVisible()
        await expect(canvas.getByText('DDNS Generated Prefix')).toBeVisible()
        await expect(canvas.getByText('myhost')).toBeVisible()
        await expect(canvas.getByText('DDNS Override Client Update')).toBeVisible()
        await expect(canvas.getByText('true')).toBeVisible()

        // Excluded parameters from the configuration must not be shown.
        await expect(canvas.queryByText('Client Classes')).toBeNull()
        await expect(canvas.queryByText('Option Data')).toBeNull()
        await expect(canvas.queryByText('Control Sockets')).toBeNull()

        await expect(canvas.getByRole('group', { name: 'Global DHCP Options' })).toBeVisible()
        await expect(canvas.getByText('(5) Name Server')).toBeVisible()
        await expect(canvas.getByText('always sent')).toBeVisible()
        await expect(canvas.getByText('192.0.2.2')).toBeVisible()
    },
}

export const Empty: Story = {
    args: {
        dhcpParameters: [],
        dhcpOptions: [[]],
        editBegin: fn(),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)

        await expect(canvas.getByRole('button', { name: 'Edit' })).toBeEnabled()
        await expect(canvas.getByRole('group', { name: 'Global DHCP Parameters' })).toBeVisible()
        await expect(canvas.getByRole('group', { name: 'Global DHCP Options' })).toBeVisible()
        await expect(canvas.getByText('No parameters configured.')).toBeVisible()
        await expect(canvas.getByText('No options configured.')).toBeVisible()
    },
}
