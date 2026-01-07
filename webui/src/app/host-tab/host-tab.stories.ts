import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfirmationService, MessageService } from 'primeng/api'
import { HostTabComponent } from './host-tab.component'
import { toastDecorator } from '../utils-stories'
import { provideRouter, withHashLocation } from '@angular/router'

export default {
    title: 'App/HostTab',
    component: HostTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter(
                    [
                        { path: 'dhcp/hosts/:id', component: HostTabComponent },
                        { path: '**', component: HostTabComponent },
                    ],
                    withHashLocation()
                ),
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/leases?hostId=1',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    items: [],
                    conflicts: 0,
                    erredApps: 0,
                },
            },
        ],
    },
} as Meta

type Story = StoryObj<HostTabComponent>

export const ViewDhcpv4Host: Story = {
    args: {
        host: {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '51:52:53:54:55:56',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.23',
                },
            ],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                    clientClasses: ['access-point', 'router', 'cable-modem'],
                    nextServer: '192.0.2.2',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/bootfile',
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    clientClasses: ['access-point', 'router', 'cable-modem'],
                    nextServer: '192.0.2.2',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/bootfile',
                },
            ],
        },
    },
}

export const ViewDhcpv6Host: Story = {
    args: {
        host: {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
                {
                    idType: 'hw-address',
                    idHexValue: '51:52:53:54:55:56',
                },
            ],
            addressReservations: [
                {
                    address: '2001:db8:1::1',
                },
                {
                    address: '2001:db8:1::2',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8:2::/64',
                },
                {
                    address: '2001:db8:3::/64',
                },
            ],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                    clientClasses: ['access-point', 'router', 'cable-modem'],
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    clientClasses: ['access-point', 'router', 'cable-modem'],
                },
            ],
        },
    },
}
