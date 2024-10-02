import { Meta, StoryObj, applicationConfig, moduleMetadata } from '@storybook/angular'
import { MessageModule } from 'primeng/message'
import { PanelModule } from 'primeng/panel'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { TooltipModule } from 'primeng/tooltip'
import { HaStatusComponent } from './ha-status.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ServicesService, ServicesStatus } from '../backend'
import { MessageService } from 'primeng/api'
import { importProvidersFrom } from '@angular/core'
import { HttpClientModule } from '@angular/common/http'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ToastModule } from 'primeng/toast'
import { RouterTestingModule } from '@angular/router/testing'
import { TagModule } from 'primeng/tag'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ProgressBarModule } from 'primeng/progressbar'
import { TableModule } from 'primeng/table'
import { ButtonModule } from 'primeng/button'

let mockHubAndSpokeStatus: ServicesStatus = {
    items: [
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server1',
                    primaryServer: {
                        age: 0,
                        appId: 234,
                        controlAddress: '192.0.2.1:8080',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server1'],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                    secondaryServer: {
                        age: 0,
                        appId: 123,
                        controlAddress: '192.0.2.2:8080',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'standby',
                        scopes: [],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                },
            },
        },
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server3',
                    primaryServer: {
                        age: 0,
                        appId: 345,
                        controlAddress: '192.0.2.3:8080',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server3'],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
                    },
                    secondaryServer: {
                        age: 0,
                        appId: 123,
                        controlAddress: '192.0.2.2:8081',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'standby',
                        scopes: [],
                        state: 'hot-standby',
                        statusTime: '2024-02-16',
                        commInterrupted: 1,
                        connectingClients: 5,
                        unackedClients: 1,
                        unackedClientsLeft: 4,
                        analyzedPackets: 0,
                    },
                },
            },
        },
    ],
}

let mockPassiveBackupStatus: ServicesStatus = {
    items: [
        {
            status: {
                daemon: 'dhcp4',
                haServers: {
                    relationship: 'server1',
                    primaryServer: {
                        age: 0,
                        appId: 234,
                        controlAddress: '192.0.2.1:8080',
                        failoverTime: null,
                        id: 1,
                        inTouch: true,
                        role: 'primary',
                        scopes: ['server1'],
                        state: 'passive-backup',
                        statusTime: '2024-02-16',
                    },
                },
            },
        },
    ],
}

export default {
    title: 'App/HaStatus',
    component: HaStatusComponent,
    decorators: [
        applicationConfig({
            providers: [
                importProvidersFrom(HttpClientModule),
                importProvidersFrom(NoopAnimationsModule),
                ServicesService,
                MessageService,
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                PanelModule,
                TooltipModule,
                MessageModule,
                OverlayPanelModule,
                ProgressBarModule,
                ProgressSpinnerModule,
                RouterTestingModule,
                TableModule,
                TagModule,
                ToastModule,
            ],
            declarations: [
                EntityLinkComponent,
                HaStatusComponent,
                HaStatusComponent,
                HelpTipComponent,
                LocaltimePipe,
                PlaceholderPipe,
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/apps/123/services/status',
                method: 'GET',
                status: 200,
                delay: 200,
                response: mockHubAndSpokeStatus,
            },
            {
                url: 'http://localhost/api/apps/234/services/status',
                method: 'GET',
                status: 200,
                delay: 200,
                response: mockPassiveBackupStatus,
            },
        ],
    },
} as Meta

type Story = StoryObj<HaStatusComponent>

export const hubAndSpoke: Story = {
    args: {
        appId: 123,
        daemonName: 'dhcp4',
    },
}

export const passiveBackup: Story = {
    args: {
        appId: 234,
        daemonName: 'dhcp4',
    },
}
