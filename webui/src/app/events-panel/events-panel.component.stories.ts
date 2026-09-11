import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter, withHashLocation } from '@angular/router'
import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { ConfirmationService, MessageService } from '@openng/optimus-ui/api'
import { Events } from '../backend'
import { toastDecorator } from '../utils-stories'
import { EventsPanelComponent } from './events-panel.component'
import { action } from 'storybook/actions'
import { userEvent, within, expect } from 'storybook/test'

export default {
    title: 'App/EventsPanel',
    component: EventsPanelComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter([{ path: '**', component: EventsPanelComponent }], withHashLocation()),
                ConfirmationService,
            ],
        }),
        toastDecorator,
    ],
    argTypes: {
        ui: {
            control: 'radio',
            options: ['bare', 'table'],
        },
    },
    args: {
        ui: 'bare',
    },
} as Meta

type Story = StoryObj<EventsPanelComponent>

const users = [
    {
        authenticationMethodId: 'internal',
        groups: [1],
        id: 1,
        lastname: 'admin',
        login: 'admin',
        name: 'admin',
    },
    {
        authenticationMethodId: 'oidc',
        email: 'user@example.org',
        externalId: '123',
        groups: [1],
        id: 2,
    },
    {
        authenticationMethodId: 'ldap',
        externalId: '234',
        groups: [1],
        id: 3,
    },
    {
        authenticationMethodId: 'ldap',
        externalId: '2345',
        groups: [1],
        id: 4,
        lastname: 'admin',
        login: 'admin',
        name: 'admin',
    },
]

export const Primary: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/events?start=0&limit=10&level=0',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: (request) => {
                    const { searchParams } = request
                    const limit = parseInt(searchParams.limit, 10)
                    const start = parseInt(searchParams.start, 10)
                    action('onFetchEvents')()
                    return {
                        total: 100,
                        items: Array(limit)
                            .fill(null)
                            .map((_, idx) => ({
                                id: start + idx,
                                createdAt: new Date().toLocaleString(),
                                details:
                                    idx % 5 !== 1
                                        ? null
                                        : Array(start + idx)
                                              .fill('Lorem ipsum.')
                                              .join(' '),
                                level: idx % 4 == 3 ? undefined : idx % 4,
                                text: Array(10)
                                    .fill(0)
                                    .map(
                                        () =>
                                            ['Lorem', 'ipsum', 'dolor', 'sit', 'ament.'][Math.round(Math.random() * 4)]
                                    )
                                    .join(' '),
                            })),
                    } as Events
                },
            },
            {
                url: 'api/users?start=s&limit=l',
                method: 'GET',
                status: 200,
                delay: 500,
                response: () => ({
                    items: users,
                    total: users.length,
                }),
            },
            {
                url: 'api/machines/directory',
                method: 'GET',
                status: 200,
                delay: 500,
                response: () => ({
                    items: [
                        { address: 'agent-kea', id: 7 },
                        { address: 'agent-kea6', id: 1 },
                    ],
                    total: 2,
                }),
            },
        ],
    },
}

export const Empty: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/events?start=0&limit=10&level=0',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    items: [],
                    total: 0,
                } as Events,
            },
        ],
    },
}

export const TestUsersDropdown: Story = {
    globals: {
        role: 'super-admin',
    },
    args: {
        ui: 'table',
    },
    parameters: Primary.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        // Configure delay between consecutive user events to be more human-like and to give more time for OptimusUI animations when automatically testing.
        const user = userEvent.setup({ delay: 250 })
        const dropdowns = await canvas.findAllByRole('combobox', { name: 'any' })
        await expect(dropdowns).toBeTruthy()
        await expect(dropdowns.length).toEqual(3)

        // Act
        await user.click(dropdowns[2]) // The last combobox is expected to be the Users dropdown.

        // Assert
        const options = await canvas.findAllByRole('option')
        await expect(options).toBeTruthy()
        await expect(options.length).toEqual(users.length)
        await expect(options[0]).toHaveTextContent('admin (internal)')
        // This user has no login, so email address is used to label the user.
        await expect(options[1]).toHaveTextContent('user@example.org (OIDC)')
        // This user has no login and no email, so "unknown" label should be displayed.
        await expect(options[2]).toHaveTextContent('unknown (LDAP)')
        await expect(options[3]).toHaveTextContent('admin (LDAP)')
    },
}
