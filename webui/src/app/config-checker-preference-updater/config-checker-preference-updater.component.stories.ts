import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideAnimations } from '@angular/platform-browser/animations'
import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { ConfigChecker, ConfigCheckerPreferences, ConfigCheckers, ServicesService } from '../backend'
import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater.component'
import { action } from '@storybook/addon-actions'
import { toastDecorator } from '../utils-stories'

const mockData: ConfigCheckers = {
    items: [
        {
            name: 'out_of_pool_reservation',
            selectors: ['each-daemon', 'kea-daemon'],
            state: ConfigChecker.StateEnum.Disabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: false,
        },
        {
            name: 'dispensable_subnet',
            selectors: ['each-daemon'],
            state: ConfigChecker.StateEnum.Enabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: true,
        },
    ],
    total: 2,
}

export default {
    title: 'App/ConfigCheckerPreferenceUpdater',
    component: ConfigCheckerPreferenceUpdaterComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                ServicesService,
                provideAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
            ],
        }),
        toastDecorator,
    ],
    argTypes: {
        daemonID: {
            type: { name: 'number', required: false },
        },
        minimal: {
            type: 'boolean',
            defaultValue: false,
        },
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'GET',
                status: 200,
                response: mockData,
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'PUT',
                status: 200,
                delay: 2000,
                response: (request) => {
                    const { body } = request
                    const preferences: ConfigCheckerPreferences = JSON.parse(body)
                    action('onUpdatePreferences')(preferences.items)

                    for (let preference of preferences.items) {
                        for (let checker of mockData.items) {
                            if (preference.name === checker.name) {
                                checker.state = preference.state
                            }
                        }
                    }
                    return mockData
                },
            },
        ],
    },
} as Meta

type Story = StoryObj<ConfigCheckerPreferenceUpdaterComponent>

export const GlobalCheckers: Story = {
    args: {
        daemonID: null,
    },
}

export const DaemonCheckers: Story = {
    args: {
        daemonID: 1,
    },
}
