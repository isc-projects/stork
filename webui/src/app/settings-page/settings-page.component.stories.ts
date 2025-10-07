import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { SettingsPageComponent } from './settings-page.component'
import { provideAnimations, provideNoopAnimations } from '@angular/platform-browser/animations'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { FieldsetModule } from 'primeng/fieldset'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { MessagesModule } from 'primeng/messages'
import { PopoverModule } from 'primeng/popover'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { DividerModule } from 'primeng/divider'
import { Settings } from '../backend'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { CheckboxModule } from 'primeng/checkbox'
import { InputNumberModule } from 'primeng/inputnumber'
import { InputTextModule } from 'primeng/inputtext'
import { provideRouter, RouterModule } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'

let mockGetSettingsResponse: Settings = {
    bind9StatsPullerInterval: 10,
    grafanaUrl: 'http://grafana.org',
    grafanaDhcp4DashboardId: 'dhcp4',
    grafanaDhcp6DashboardId: 'dhcp6',
    keaHostsPullerInterval: 12,
    keaStatsPullerInterval: 15,
    keaStatusPullerInterval: 23,
    appsStatePullerInterval: 44,
    enableMachineRegistration: true,
}

export default {
    title: 'App/SettingsPage',
    component: SettingsPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideAnimations(),
                provideRouter([]),
            ],
        }),
        moduleMetadata({
            imports: [
                BreadcrumbModule,
                ButtonModule,
                CheckboxModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                MessagesModule,
                PopoverModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
                RouterModule,
                ToastModule,
                InputNumberModule,
                InputTextModule,
            ],
            declarations: [BreadcrumbsComponent, HelpTipComponent, SettingsPageComponent],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/settings',
                method: 'GET',
                status: 200,
                delay: 1000,
                response: mockGetSettingsResponse,
            },
            {
                url: 'http://localhost/api/settings',
                method: 'PUT',
                status: 200,
                delay: 0,
                response: {},
            },
        ],
    },
} as Meta

type Story = StoryObj<SettingsPageComponent>

export const SettingsForm: Story = {}
