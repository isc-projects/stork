import { HostFormComponent } from './host-form.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { provideAnimations } from '@angular/platform-browser/animations'
import { UntypedFormBuilder } from '@angular/forms'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { CreateHostBeginResponse, DHCPService, UpdateHostBeginResponse } from '../backend'
import { provideRouter } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'

const mockCreateHostBeginData: CreateHostBeginResponse = {
    id: 123,
    subnets: [
        {
            id: 1,
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    daemonId: 1,
                },
                {
                    daemonId: 2,
                },
            ],
        },
        {
            id: 2,
            subnet: '192.0.3.0/24',
            localSubnets: [
                {
                    daemonId: 2,
                },
                {
                    daemonId: 3,
                },
            ],
        },
        {
            id: 3,
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    daemonId: 4,
                },
            ],
        },
        {
            id: 4,
            subnet: '2001:db8:2::/64',
            localSubnets: [
                {
                    daemonId: 5,
                },
            ],
        },
    ],
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
        },
    ],
    clientClasses: ['router', 'cable-modem', 'access-point'],
}

let mockUpdateHostBeginData: UpdateHostBeginResponse = mockCreateHostBeginData
mockUpdateHostBeginData.host = {
    id: 123,
    subnetId: 1,
    subnetPrefix: '192.0.2.0/24',
    hostIdentifiers: [
        {
            idType: 'hw-address',
            idHexValue: '01:02:03:04:05:06',
        },
    ],
    addressReservations: [
        {
            address: '192.0.2.4',
        },
    ],
    prefixReservations: [],
    hostname: 'foo.example.org',
    localHosts: [
        {
            daemonId: 1,
            dataSource: 'api',
            nextServer: '192.2.2.1',
            serverHostname: 'server1.example.org',
            bootFileName: '/tmp/boot1',
            clientClasses: ['router', 'switch'],
        },
        {
            daemonId: 2,
            dataSource: 'api',
            nextServer: '192.2.2.1',
            serverHostname: 'server2.example.org',
            bootFileName: '/tmp/boot1',
            clientClasses: ['access-point', 'router'],
        },
    ],
}

export default {
    title: 'App/HostForm',
    component: HostFormComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting(), provideRouter([])],
        }),
        moduleMetadata({
            providers: [UntypedFormBuilder, DHCPService, MessageService, provideAnimations()],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/hosts/new/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockCreateHostBeginData,
            },
            {
                url: 'http://localhost/api/hosts/:id/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockUpdateHostBeginData,
            },
        ],
    },
} as Meta

type Story = StoryObj<HostFormComponent>

export const NewHost: Story = {
    args: {},
}

export const UpdatedHost: Story = {
    args: {
        hostId: 123,
    },
}
