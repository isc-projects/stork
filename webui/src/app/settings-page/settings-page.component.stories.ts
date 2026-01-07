import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { SettingsPageComponent } from './settings-page.component'
import { MessageService } from 'primeng/api'
import { Settings } from '../backend'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { toastDecorator } from '../utils-stories'
import { provideRouter, withHashLocation } from '@angular/router'

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
                provideRouter([{ path: '**', component: SettingsPageComponent }], withHashLocation()),
            ],
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
