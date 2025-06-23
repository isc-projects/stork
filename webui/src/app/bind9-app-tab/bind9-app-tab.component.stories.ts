import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { of } from 'rxjs'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { PanelModule } from 'primeng/panel'
import { AppOverviewComponent } from '../app-overview/app-overview.component'
import { TabViewModule } from 'primeng/tabview'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { provideRouter, RouterModule } from '@angular/router'
import { FormsModule } from '@angular/forms'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { EventTextComponent } from '../event-text/event-text.component'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { TooltipModule } from 'primeng/tooltip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
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

const mockAppTab = {
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

const versionServiceStub = {
    sanitizeSemver: () => '9.18.30',
    getCurrentData: () => of({} as AppsVersions),
    getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
}

export default {
    title: 'App/Bind9AppTab',
    component: Bind9AppTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideNoopAnimations(),
                provideRouter([{ path: 'iframe.html', component: Bind9AppTabComponent }]),
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
                TabViewModule,
                TooltipModule,
                PanelModule,
                OverlayPanelModule,
                DataViewModule,
                TableModule,
                MockManagedAccessDirective,
                ToastModule,
                ProgressSpinnerModule,
            ],
            declarations: [
                AppOverviewComponent,
                Bind9AppTabComponent,
                DaemonNiceNamePipe,
                DurationPipe,
                LocaltimePipe,
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
        ],
    },
} as Meta

type Story = StoryObj<Bind9AppTabComponent>

export const Default: Story = {
    args: {
        appTab: mockAppTab,
    },
}
