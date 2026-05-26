import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { LoginScreenComponent } from './login-screen.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AuthenticationMethod, AuthenticationMethods, Version } from '../backend'
import { MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { action } from 'storybook/actions'
import { provideRouter, withHashLocation } from '@angular/router'
import { userEvent, within, expect, waitFor } from 'storybook/test'

export default {
    title: 'App/LoginScreen',
    component: LoginScreenComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter([{ path: '**', component: LoginScreenComponent }], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/version',
                method: 'GET',
                status: 200,
                delay: 1000,
                response: { date: '2023-03-03', version: '1.4.2' } as Version,
            },
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    total: 5,
                    items: [
                        {
                            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
                            formLabelIdentifier: 'Username/Email',
                            formLabelSecret: 'Password',
                            id: 'internal',
                            name: 'Internal',
                        },
                        {
                            description: 'Fusce faucibus mauris purus, in mattis tellus pellentesque sed.',
                            formLabelIdentifier: 'Username',
                            formLabelSecret: 'Password',
                            id: 'ldap',
                            name: 'LDAP',
                        },
                        {
                            description:
                                'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Fusce eleifend condimentum accumsan. Fusce faucibus mauris purus, in mattis tellus pellentesque sed. Quisque congue eu lacus ut ultrices. Nulla consectetur commodo ante sed blandit. Suspendisse sollicitudin nisl ac ipsum maximus rhoncus. Sed dolor massa, dignissim in luctus at, pulvinar nec mauris. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Fusce eleifend condimentum accumsan. Fusce faucibus mauris purus, in mattis tellus pellentesque sed. Quisque congue eu lacus ut ultrices. Nulla consectetur commodo ante sed blandit. Suspendisse sollicitudin nisl ac ipsum maximus rhoncus. Sed dolor massa, dignissim in luctus at, pulvinar nec mauris. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Fusce eleifend condimentum accumsan. Fusce faucibus mauris purus, in mattis tellus pellentesque sed. Quisque congue eu lacus ut ultrices. Nulla consectetur commodo ante sed blandit. Suspendisse sollicitudin nisl ac ipsum maximus rhoncus. Sed dolor massa, dignissim in luctus at, pulvinar nec mauris.',
                            formLabelIdentifier: 'Lorem',
                            formLabelSecret: 'Ipsum',
                            id: 'dolor',
                            name: 'Sit amet',
                        },
                        {
                            description: 'Fusce eleifend condimentum accumsan.',
                            id: 'passwordless',
                            name: 'Passwordless',
                        },
                        {
                            description:
                                'OAuth2/OIDC authentication. You will get redirected to OpenID Provider to authenticate and authorize your access to Stork.',
                            id: 'oidc',
                            name: 'Log in with OpenID Connect',
                        },
                    ],
                } as AuthenticationMethods,
            },
        ],
    },
} as Meta

type Story = StoryObj<LoginScreenComponent>

export const Primary: Story = {}

export const Loading: Story = {
    parameters: {
        mockData: [],
    },
}

const items: AuthenticationMethod[] = Array(40)
    .fill(0)
    .map((_, i) => ({
        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
        formLabelIdentifier: `Identifier ${i}`,
        formLabelSecret: `Secret ${i}`,
        id: i % 4 == 0 && i < 20 ? 'internal' : i.toString(),
        name: `Method ${i}`,
    }))

export const ManyButtons: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 200,
                delay: 0,
                response: {
                    total: items.length,
                    items,
                } as AuthenticationMethods,
            },
        ],
    },
}

export const SingleMethod: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 200,
                delay: 0,
                response: {
                    total: 4,
                    items: [
                        {
                            description: 'LDAP',
                            formLabelIdentifier: 'Username',
                            formLabelSecret: 'Password',
                            id: 'ldap',
                            name: 'LDAP',
                        },
                    ],
                } as AuthenticationMethods,
            },
        ],
    },
}

export const FailedFetch: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 500,
                delay: 500,
                response: () => {
                    action('onFetchAuthenticationMethods')()
                },
            },
        ],
    },
}

export const TestAuthMethodIsStoredInLocalStorage: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/version',
                method: 'GET',
                status: 200,
                delay: 100,
                response: { date: '2023-03-03', version: '1.4.2' } as Version,
            },
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 200,
                delay: 100,
                response: {
                    total: 3,
                    items: [
                        {
                            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
                            formLabelIdentifier: 'Username/Email',
                            formLabelSecret: 'Password',
                            id: 'internal',
                            name: 'Internal',
                        },
                        {
                            description: 'Fusce faucibus mauris purus, in mattis tellus pellentesque sed.',
                            formLabelIdentifier: 'Username',
                            formLabelSecret: 'Password',
                            id: 'ldap',
                            name: 'LDAP',
                        },
                        {
                            description:
                                'OAuth2/OIDC authentication. You will get redirected to OpenID Provider to authenticate and authorize your access to Stork.',
                            id: 'oidc',
                            name: 'Log in with OpenID Connect',
                        },
                    ],
                } as AuthenticationMethods,
            },
        ],
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const combobox = await canvas.findByRole('combobox')
        // Configure delay between consecutive user events to be more human-like and to give more time for PrimeNG animations when automatically testing.
        const user = userEvent.setup({ delay: 50 })

        // Act
        let options
        await user.click(combobox)
        options = await canvas.findAllByRole('option')
        await expect(options.length).toBeGreaterThanOrEqual(3)

        // Reset storage to have selected the first option.
        await user.click(options[0])
        await expect(combobox).toHaveTextContent('Internal')

        await user.click(combobox)
        options = await canvas.findAllByRole('option')
        await expect(options.length).toBeGreaterThanOrEqual(3)
        await user.click(options[2])

        // Assert
        const stored = localStorage.getItem('selected-auth-method')
        await expect(stored).not.toBeFalsy()
        await expect(stored).toEqual('oidc')

        await expect(combobox).toHaveTextContent('Log in with OpenID Connect')
        await expect(canvas.queryByLabelText('Username')).toBeNull()
        await expect(canvas.queryByLabelText('Username/Email')).toBeNull()
        await expect(canvas.queryByLabelText('Password')).toBeNull()
        const btn = await canvas.findByRole('link', { name: 'Log in with OpenID Connect' })
        await expect(btn).not.toBeFalsy()
    },
}

export const TestAuthMethodIsReadFromLocalStorage: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/version',
                method: 'GET',
                status: 200,
                delay: 100,
                response: { date: '2023-03-03', version: '1.4.2' } as Version,
            },
            {
                url: 'http://localhost/api/authentication-methods',
                method: 'GET',
                status: 200,
                delay: 200,
                response: {
                    total: 3,
                    items: [
                        {
                            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
                            formLabelIdentifier: 'Username/Email',
                            formLabelSecret: 'Password',
                            id: 'internal',
                            name: 'Internal',
                        },
                        {
                            description: 'Fusce faucibus mauris purus, in mattis tellus pellentesque sed.',
                            formLabelIdentifier: 'Username',
                            formLabelSecret: 'Password',
                            id: 'ldap',
                            name: 'LDAP',
                        },
                        {
                            description:
                                'OAuth2/OIDC authentication. You will get redirected to OpenID Provider to authenticate and authorize your access to Stork.',
                            id: 'oidc',
                            name: 'Log in with OpenID Connect',
                        },
                    ],
                } as AuthenticationMethods,
            },
        ],
    },
    play: async ({ canvasElement }) => {
        // Arrange + Act + Assert
        localStorage.setItem('selected-auth-method', 'ldap')
        const canvas = within(canvasElement)
        await waitFor(() => canvas.findByRole('combobox'))
        const combobox = await canvas.findByRole('combobox')
        await expect(combobox).toHaveTextContent('LDAP')
    },
}

export const TestDisplayFirstAuthMethodIfLocalStorageEntryNotFound: Story = {
    parameters: TestAuthMethodIsReadFromLocalStorage.parameters,
    play: async ({ canvasElement }) => {
        // Arrange + Act + Assert
        localStorage.setItem('selected-auth-method', 'does-not-exist-any-more')
        const canvas = within(canvasElement)
        await waitFor(() => canvas.findByRole('combobox'))
        const combobox = await canvas.findByRole('combobox')
        await expect(combobox).toHaveTextContent('Internal')
    },
}
