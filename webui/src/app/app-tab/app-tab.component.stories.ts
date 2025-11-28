import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AppTabComponent } from './app-tab.component'
import { of } from 'rxjs'
import { provideRouter } from '@angular/router'
import { AppsVersions } from '../backend'
import { Severity, VersionService } from '../version.service'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'

const mockBind9AppTab = {
    app: {
        id: 1,
        name: 'bind9@bind9-app-tab',
        type: 'bind9',
        accessPoints: [],
        version: '1.0.0',
        machine: {
            id: 1,
            address: '127.0.0.1',
            hostname: 'test',
        },
        details: {
            daemon: {
                name: 'named',
                id: 1,
                pid: 1,
                active: true,
                monitored: true,
                version: '1.0.0',
                uptime: 100,
                reloadedAt: '2021-01-01',
                zoneCount: 1,
                views: [],
                agentCommErrors: 0,
                rndcCommErrors: 0,
                statsCommErrors: 0,
            },
        },
    },
}

const mockPowerDNSAppTabNoURLs = {
    app: {
        id: 1,
        name: 'pdns@pdns-app-tab',
        type: 'pdns',
        accessPoints: [],
        version: '4.1.2',
        machine: {
            id: 1,
            address: '127.0.0.1',
            hostname: 'test',
        },
        details: {
            pdnsDaemon: {
                name: 'pdns',
                id: 1,
                pid: 1,
                active: true,
                monitored: true,
                version: '4.1.2',
                uptime: 100,
            },
        },
    },
}

const mockPowerDNSAppTab = {
    app: {
        id: 1,
        name: 'pdns@pdns-app-tab',
        type: 'pdns',
        accessPoints: [],
        version: '4.1.2',
        machine: {
            id: 1,
            address: '127.0.0.1',
            hostname: 'test',
        },
        details: {
            pdnsDaemon: {
                name: 'pdns',
                id: 1,
                pid: 1,
                active: true,
                monitored: true,
                version: '4.1.2',
                uptime: 100,
                url: 'http://localhost:5380',
                configUrl: 'http://localhost:5380/config',
                zonesUrl: 'http://localhost:5380/zones',
                autoprimariesUrl: 'http://localhost:5380/autoprimaries',
            },
        },
    },
}

const versionServiceStub = {
    sanitizeSemver: () => '9.18.30',
    getCurrentData: () => of({} as AppsVersions),
    getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
}

const mockShortConfigResponse = {
    files: [
        {
            sourcePath: '/etc/bind/test.conf',
            fileType: 'config',
            contents: ['options {', '\tlisten-on {', '\t\t127.0.0.1;', '\t};', '};'],
        },
    ],
}

const mockLongConfigResponse = {
    files: [
        {
            sourcePath: '/etc/bind/test.conf',
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
    title: 'App/AppTab',
    component: AppTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideAnimations(),
                provideRouter([{ path: 'iframe.html', component: AppTabComponent }]),
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/machines/directory',
                method: 'GET',
                status: 200,
                response: { items: [] },
            },
            {
                url: 'http://localhost/apps/directory',
                method: 'GET',
                status: 200,
                response: { items: [] },
            },
            {
                url: 'http://localhost/apps/:id/name',
                method: 'PUT',
                status: 200,
                response: {},
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?filter=config&fileSelector=config',
                method: 'GET',
                status: 200,
                response: mockShortConfigResponse,
                delay: 1000,
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/bind9-config?fileSelector=config',
                method: 'GET',
                status: 200,
                response: mockLongConfigResponse,
                delay: 3000,
            },
        ],
    },
} as Meta

type Story = StoryObj<AppTabComponent>

export const Bind9: Story = {
    args: {
        appTab: mockBind9AppTab,
    },
}

export const PowerDNS: Story = {
    args: {
        appTab: mockPowerDNSAppTab,
    },
}

export const PowerDNSNoURLs: Story = {
    args: {
        appTab: mockPowerDNSAppTabNoURLs,
    },
}
