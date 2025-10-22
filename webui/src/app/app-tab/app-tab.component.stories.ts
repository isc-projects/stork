import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AppTabComponent } from './app-tab.component'
import { of } from 'rxjs'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { PanelModule } from 'primeng/panel'
import { AppOverviewComponent } from '../app-overview/app-overview.component'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { provideRouter, RouterModule } from '@angular/router'
import { FormsModule } from '@angular/forms'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { EventTextComponent } from '../event-text/event-text.component'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { TooltipModule } from 'primeng/tooltip'
import { PopoverModule } from 'primeng/popover'
import { DataViewModule } from 'primeng/dataview'
import { DurationPipe } from '../pipes/duration.pipe'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { AppsVersions } from '../backend'
import { Severity, VersionService } from '../version.service'
import { Directive, Input, Output, EventEmitter, AfterViewInit } from '@angular/core'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { Bind9DaemonComponent } from '../bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from '../pdns-daemon/pdns-daemon.component'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { Bind9DaemonControlsComponent } from '../bind9-daemon-controls/bind9-daemon-controls.component'
import { Bind9ConfigPreviewComponent } from '../bind9-config-preview/bind9-config-preview.component'

// Mock directive that always grants access
@Directive({
    selector: '[appAccessEntity]',
    standalone: true,
})
class MockManagedAccessDirective implements AfterViewInit {
    @Input() appAccessEntity: any
    @Input() appAccessType: any = 'read'
    @Input() appHideIfNoAccess: boolean = false
    @Output() appHasAccess = new EventEmitter<boolean>()

    ngAfterViewInit() {
        this.appHasAccess.emit(true)
    }
}

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
                provideNoopAnimations(),
                provideRouter([{ path: 'iframe.html', component: AppTabComponent }]),
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                DialogModule,
                FormsModule,
                RouterModule,
                TooltipModule,
                PanelModule,
                PopoverModule,
                DataViewModule,
                TableModule,
                MockManagedAccessDirective,
                ToastModule,
                ProgressSpinnerModule,
                TabViewComponent,
                ConfirmDialogModule,
                Bind9DaemonControlsComponent,
                Bind9ConfigPreviewComponent,
            ],
            declarations: [
                AppOverviewComponent,
                AppTabComponent,
                Bind9DaemonComponent,
                DaemonNiceNamePipe,
                DurationPipe,
                LocaltimePipe,
                PdnsDaemonComponent,
                PlaceholderPipe,
                RenameAppDialogComponent,
                AppOverviewComponent,
                EventsPanelComponent,
                EventTextComponent,
                VersionStatusComponent,
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
