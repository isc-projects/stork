import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { Bind9ConfigViewFeederComponent } from './bind9-config-view-feeder.component'
import { provideAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'

const rndcKeyResponse = {
    files: [
        {
            sourcePath: '/etc/bind/rndc.key',
            fileType: 'config',
            contents: [
                'key "rndc-key" {',
                '\talgorithm hmac-sha256;',
                '\tsecret "UlJY3N2FdJ5cWUT6jQt/OPEnT9ap4b45Pzo1724yYw=";',
                '};',
            ],
        },
    ],
}

const configResponse = {
    files: [
        {
            sourcePath: '/etc/bind/named.conf',
            fileType: 'config',
            contents: [
                'options {',
                '\tlisten-on {',
                '\t\t127.0.0.1;',
                '\t};',
                '};',
                'view "internal" {',
                '\tzone "internal" {',
                '\t\ttype primary;',
                '\t\tfile "internal.zone";',
                '\t};',
                '};',
                'view "external" {',
                '\tzone "external" {',
                '\t\ttype primary;',
                '\t\tfile "external.zone";',
                '\t};',
                '};',
            ],
        },
    ],
}
export default {
    title: 'App/Bind9ConfigViewFeeder',
    component: Bind9ConfigViewFeederComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService, provideAnimations(), provideHttpClient(withInterceptorsFromDi())],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<Bind9ConfigViewFeederComponent>

export const ViewConfig: Story = {
    args: {
        daemonId: 1,
        fileType: 'config',
        // This flag triggers the HTTP call to get the configuration. Use the
        // controls in the storybook panel to trigger the HTTP call. Flip this
        // value to true.
        active: false,
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?filter=config&fileSelector=config',
                method: 'GET',
                status: 200,
                delay: 3000,
                response: configResponse,
            },
        ],
    },
}

export const ViewRndcKey: Story = {
    args: {
        daemonId: 1,
        fileType: 'rndc-key',
        // This flag triggers the HTTP call to get the configuration. Use the
        // controls in the storybook panel to trigger the HTTP call. Flip this
        // value to true.
        active: false,
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?filter=config&fileSelector=rndc-key',
                method: 'GET',
                status: 200,
                delay: 3000,
                response: rndcKeyResponse,
            },
        ],
    },
}

export const HttpError: Story = {
    args: {
        daemonId: 1,
        fileType: 'config',
        // This flag triggers the HTTP call to get the configuration. Use the
        // controls in the storybook panel to trigger the HTTP call. Flip this
        // value to true.
        active: false,
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?filter=config&fileSelector=rndc-key',
                method: 'GET',
                status: 500,
                delay: 3000,
                response: {
                    message: 'Error getting BIND 9 configuration',
                },
            },
        ],
    },
}
