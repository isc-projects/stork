import { HttpClientTestingModule } from '@angular/common/http/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { Meta, moduleMetadata, Story } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { DialogModule } from 'primeng/dialog'
import { DividerModule } from 'primeng/divider'
import { PaginatorModule } from 'primeng/paginator'
import { PanelModule } from 'primeng/panel'
import { TagModule } from 'primeng/tag'
import { ConfigChecker, ConfigCheckerPreferences, ConfigCheckers, ConfigReports, ServicesService } from '../backend'
import { EventTextComponent } from '../event-text/event-text.component'
import { LocaltimePipe } from '../localtime.pipe'
import { ConfigReviewPanelComponent } from './config-review-panel.component'
import mockAddon from 'storybook-addon-mock'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HttpClientModule } from '@angular/common/http'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { action } from '@storybook/addon-actions'
import { ToastModule } from 'primeng/toast'
import { toastDecorator } from '../utils.stories'

const mockPreferencesData: ConfigCheckers = {
    items: [
        {
            name: 'reservations_out_of_pool',
            selectors: ['each-daemon', 'kea-daemon'],
            state: ConfigChecker.StateEnum.Disabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: false,
        },
        {
            name: 'subnet_dispensable',
            selectors: ['each-daemon'],
            state: ConfigChecker.StateEnum.Enabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: true,
        },
    ],
    total: 2,
}

export default {
    title: 'App/ConfigReviewPanel',
    component: ConfigReviewPanelComponent,
    decorators: [
        moduleMetadata({
            imports: [
                ButtonModule,
                DividerModule,
                HttpClientModule,
                NoopAnimationsModule,
                PaginatorModule,
                PanelModule,
                TagModule,
                TableModule,
                ChipModule,
                OverlayPanelModule,
                ToastModule,
            ],
            declarations: [
                ConfigReviewPanelComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                ConfigCheckerPreferencePickerComponent,
                EventTextComponent,
                LocaltimePipe,
                HelpTipComponent,
            ],
            providers: [ServicesService, MessageService],
        }),
        mockAddon,
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
                url: 'http://localhost/api/daemons/:daemonId/config-reports?start=0&limit=5',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    total: 2,
                    review: {
                        createdAt: '2022-08-25T12:34:56',
                        daemonId: 1,
                        id: 1,
                    },
                    items: [
                        {
                            checker: 'reservations_out_of_pool',
                            content: 'Something is wrong',
                            createdAt: '2022-08-25T12:34:56',
                            id: 1,
                        },
                        {
                            checker: 'subnet_dispensable',
                            content: 'Foobar',
                            createdAt: '2022-08-25T12:34:56',
                            id: 2,
                        },
                    ],
                } as ConfigReports,
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

const Template: Story<ConfigReviewPanelComponent> = (args: ConfigReviewPanelComponent) => ({
    props: args,
})

export const Primary = Template.bind({})
Primary.args = {
    daemonId: 1,
}
