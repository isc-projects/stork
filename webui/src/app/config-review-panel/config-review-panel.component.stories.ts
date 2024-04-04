import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { DividerModule } from 'primeng/divider'
import { TagModule } from 'primeng/tag'
import { ConfigChecker, ConfigCheckerPreferences, ConfigCheckers, ConfigReports, ServicesService } from '../backend'
import { EventTextComponent } from '../event-text/event-text.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ConfigReviewPanelComponent } from './config-review-panel.component'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HttpClientModule } from '@angular/common/http'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { action } from '@storybook/addon-actions'
import { ToastModule } from 'primeng/toast'
import { toastDecorator } from '../utils-stories'
import { DataViewModule } from 'primeng/dataview'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { FormsModule } from '@angular/forms'
import { importProvidersFrom } from '@angular/core'

const mockPreferencesData: ConfigCheckers = {
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
        {
            name: 'host_cmds_presence',
            selectors: ['each-daemon'],
            state: ConfigChecker.StateEnum.Enabled,
            triggers: ['manual', 'config change', 'host reservations change'],
            globallyEnabled: true,
        },
    ],
    total: 2,
}

export default {
    title: 'App/ConfigReviewPanel',
    component: ConfigReviewPanelComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService, ServicesService, importProvidersFrom(HttpClientModule)],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                DividerModule,
                HttpClientModule,
                NoopAnimationsModule,
                TagModule,
                TableModule,
                ChipModule,
                OverlayPanelModule,
                ToastModule,
                FormsModule,
                ToggleButtonModule,
                DataViewModule,
            ],
            declarations: [
                ConfigReviewPanelComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                ConfigCheckerPreferencePickerComponent,
                EventTextComponent,
                LocaltimePipe,
                HelpTipComponent,
            ],
        }),
        toastDecorator,
    ],
    argTypes: {},
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: () => {
                    action('onFetchPreferences')()
                    return mockPreferencesData
                },
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'PUT',
                status: 200,
                response: (request) => {
                    const { body } = request
                    const preferences: ConfigCheckerPreferences = JSON.parse(body)
                    action('onUpdatePreferences')(preferences.items)

                    for (let preference of preferences.items) {
                        for (let checker of mockPreferencesData.items) {
                            if (preference.name === checker.name) {
                                checker.state = preference.state
                            }
                        }
                    }
                    return mockPreferencesData
                },
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/config-reports?start=:start&limit=:limit&issuesOnly=:issuesOnly',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: (request) => {
                    const { searchParams } = request

                    let reports = Array(5)
                        .fill(0)
                        .map(() => [
                            {
                                checker: 'out_of_pool_reservation',
                                content:
                                    'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
                                createdAt: '2022-08-25T12:34:56',
                                id: 1,
                            },
                            {
                                checker: 'dispensable_subnet',
                                content:
                                    'Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
                                createdAt: '2022-08-25T12:34:56',
                                id: 2,
                            },
                            {
                                checker: 'host_cmds_presence',
                                createdAt: '2022-08-25T12:34:56',
                                id: 3,
                            },
                        ])
                        .reduce((acc, item) => [...acc, ...item], [])
                    reports.forEach((r, idx) => {
                        r.id = idx + 1
                        if (r.content) {
                            r.content = `${idx} ${r.content}`
                        }
                    })

                    const totalReports = reports.length
                    if (searchParams.issuesOnly == 'true') {
                        reports = reports.filter((r) => !!r.content)
                    }
                    const totalIssues = reports.filter((r) => !!r.content).length

                    const start = parseInt(searchParams.start, 10)
                    const limit = parseInt(searchParams.limit, 10)
                    const total = reports.length

                    reports = reports.slice(start, start + limit)

                    return {
                        total: total,
                        totalIssues: totalIssues,
                        totalReports: totalReports,
                        review: {
                            createdAt: '2022-08-25T12:34:56',
                            daemonId: 1,
                            id: 1,
                        },
                        items: reports,
                    } as ConfigReports
                },
            },
            {
                url: 'http://localhost/api/daemons/:daemonID/config-review',
                method: 'PUT',
                status: 200,
                delay: 1000,
            },
        ],
    },
} as Meta

type Story = StoryObj<ConfigReviewPanelComponent>

export const Primary: Story = {
    args: {
        daemonId: 1,
    },
}
