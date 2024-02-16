import { RouterModule } from '@angular/router'
import { Meta, Story, applicationConfig, moduleMetadata } from '@storybook/angular'
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
import { HaStatusPanelComponent } from '../ha-status-panel/ha-status-panel.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ToastModule } from 'primeng/toast'
import { RouterTestingModule } from '@angular/router/testing'

let mockServicesStatus: ServicesStatus = {
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
                        commInterrupted: 0,
                        connectingClients: 0,
                        unackedClients: 0,
                        unackedClientsLeft: 0,
                        analyzedPackets: 0,
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
                PanelModule,
                TooltipModule,
                MessageModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                RouterTestingModule,
                ToastModule,
            ],
            declarations: [HaStatusComponent, HaStatusPanelComponent, HelpTipComponent, LocaltimePipe, PlaceholderPipe],
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
                response: mockServicesStatus,
            },
        ],
    },
} as Meta

const Template: Story<HaStatusComponent> = (args: HaStatusComponent) => ({
    props: args,
})

export const hubAndSpoke = Template.bind({})
hubAndSpoke.args = {
    appId: 123,
    daemonName: 'dhcp4',
}
