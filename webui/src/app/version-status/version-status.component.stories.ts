import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { VersionStatusComponent } from './version-status.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { provideRouter, withHashLocation } from '@angular/router'
import { toastDecorator } from '../utils-stories'
import { AppsVersions } from '../backend'
import { userEvent, waitFor, within, expect } from 'storybook/test'

const softwareVersions: AppsVersions = {
    bind9: {
        currentStable: [
            {
                eolDate: '2028-07-01',
                esv: 'true',
                major: 9,
                minor: 20,
                range: '9.20.x',
                releaseDate: '2026-02-27',
                status: 'Current Stable',
                version: '9.20.20',
            },
            {
                eolDate: '2026-07-01',
                esv: 'true',
                major: 9,
                minor: 18,
                range: '9.18.x',
                releaseDate: '2026-02-27',
                status: 'Current Stable',
                version: '9.18.46',
            },
        ],
        latestDev: {
            major: 9,
            minor: 21,
            releaseDate: '2026-02-27',
            status: 'Development',
            version: '9.21.19',
        },
        latestSecure: [
            {
                major: 9,
                minor: 21,
                range: '9.21.x',
                releaseDate: '2026-01-21',
                status: 'Security update',
                version: '9.21.17',
            },
            {
                major: 9,
                minor: 20,
                range: '9.20.x',
                releaseDate: '2026-01-21',
                status: 'Security update',
                version: '9.20.18',
            },
            {
                major: 9,
                minor: 18,
                range: '9.18.x',
                releaseDate: '2026-01-21',
                status: 'Security update',
                version: '9.18.44',
            },
        ],
        sortedStableVersions: ['9.18.46', '9.20.20'],
    },
    dataSource: 'online',
    date: '2026-03-02',
    kea: {
        currentStable: [
            {
                eolDate: '2028-06-01',
                major: 3,
                minor: 0,
                range: '3.0.x',
                releaseDate: '2025-10-29',
                status: 'Current Stable',
                version: '3.0.2',
            },
            {
                eolDate: '2026-07-01',
                major: 2,
                minor: 6,
                range: '2.6.x',
                releaseDate: '2025-07-16',
                status: 'Current Stable',
                version: '2.6.4',
            },
        ],
        latestDev: {
            major: 3,
            minor: 1,
            releaseDate: '2026-02-25',
            status: 'Development',
            version: '3.1.6',
        },
        latestSecure: [
            {
                major: 3,
                minor: 1,
                range: '3.1.x',
                releaseDate: '2025-10-29',
                status: 'Security update',
                version: '3.1.3',
            },
            {
                major: 3,
                minor: 0,
                range: '3.0.x',
                releaseDate: '2025-10-29',
                status: 'Security update',
                version: '3.0.2',
            },
            {
                major: 2,
                minor: 6,
                range: '2.6.x',
                releaseDate: '2025-05-28',
                status: 'Security update',
                version: '2.6.3',
            },
            {
                major: 2,
                minor: 4,
                range: '2.4.x',
                releaseDate: '2025-05-28',
                status: 'Security update',
                version: '2.4.2',
            },
        ],
        sortedStableVersions: ['2.6.4', '3.0.2'],
    },
    stork: {
        currentStable: [
            {
                eolDate: '2027-01-01',
                major: 2,
                minor: 4,
                range: '2.4.x',
                releaseDate: '2026-02-25',
                status: 'Current Stable',
                version: '2.4.0',
            },
        ],
        latestDev: {
            major: 2,
            minor: 3,
            releaseDate: '2025-12-10',
            status: 'Development',
            version: '2.3.2',
        },
        latestSecure: [
            {
                major: 2,
                minor: 3,
                range: '2.3.x',
                releaseDate: '2025-10-15',
                status: 'Security update',
                version: '2.3.1',
            },
            {
                major: 2,
                minor: 2,
                range: '2.2.x',
                releaseDate: '2025-09-10',
                status: 'Security update',
                version: '2.2.1',
            },
        ],
        sortedStableVersions: ['2.4.0'],
    },
}

const meta: Meta<VersionStatusComponent> = {
    title: 'App/VersionStatusComponent',
    component: VersionStatusComponent,
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.0.2' },
        includeAnchor: true,
        inline: true,
        styleClass: 'ml-8',
    },
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
                provideRouter([], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'api/software-versions',
                method: 'GET',
                status: 200,
                response: softwareVersions,
            },
        ],
    },
}

export default meta
type Story = StoryObj<VersionStatusComponent>

export const StableUpToDate: Story = {}

export const StableUpdateAvail: Story = {
    args: {
        daemon: { id: 3, name: 'stork', version: '2.2.1' },
    },
}

export const StableSecurityUpdateAvail: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.0.0' },
    },
}

export const StableOld: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '2.2.0' },
    },
}

export const StableNewUnknown: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.20.0' },
    },
}

export const DevUpToDate: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.1.6' },
    },
}

export const DevUpdateAvail: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.1.5' },
    },
}

export const DevSecurityUpdateAvail: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.1.2' },
    },
}

export const DevOld: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '2.7.9' },
    },
}

export const DevNewUnknown: Story = {
    args: {
        daemon: { id: 2, name: 'dhcp4', version: '3.21.1' },
    },
}

export const BlockMessage: Story = {
    args: {
        inline: false,
    },
}

export const NoAnchor: Story = {
    args: {
        includeAnchor: false,
    },
}

export const TestFeedbackIsDisplayed: Story = {
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const body = within(canvasElement.parentElement)
        const user = userEvent.setup({ delay: 100 })

        // Act
        const anchor = await canvas.findByLabelText('More Version Info')
        await expect(anchor.childElementCount).toEqual(1)
        const tooltipIcon = anchor.children[0]
        await user.hover(tooltipIcon)

        // Assert
        await waitFor(() => expect(body.getByRole('tooltip')))
        await waitFor(() => expect(body.getByText('3.0.2 is current Kea stable version (known as of 2026-03-02).')))
    },
}
