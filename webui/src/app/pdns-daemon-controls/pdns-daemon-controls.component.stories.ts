import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { PdnsDaemonControlsComponent } from './pdns-daemon-controls.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter } from '@angular/router'
import { expect, within } from 'storybook/test'

export default {
    title: 'App/PdnsDaemonControls',
    component: PdnsDaemonControlsComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), provideRouter([])],
        }),
    ],
    parameters: {
        mockData: [],
    },
} as Meta

type Story = StoryObj<PdnsDaemonControlsComponent>

export const Primary: Story = {
    args: {
        daemonId: 1,
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
