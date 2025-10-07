import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { FormsModule } from '@angular/forms'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideRouter, RouterModule } from '@angular/router'
import { ConfirmationService, MessageService } from 'primeng/api'
import { ChipModule } from 'primeng/chip'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { FieldsetModule } from 'primeng/fieldset'
import { PopoverModule } from 'primeng/popover'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { ToastModule } from 'primeng/toast'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { TreeModule } from 'primeng/tree'
import { DhcpClientClassSetViewComponent } from '../dhcp-client-class-set-view/dhcp-client-class-set-view.component'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HostTabComponent } from './host-tab.component'
import { IdentifierComponent } from '../identifier/identifier.component'
import { toastDecorator } from '../utils-stories'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ByteCharacterComponent } from '../byte-character/byte-character.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { HostDataSourceLabelComponent } from '../host-data-source-label/host-data-source-label.component'
import { ButtonModule } from 'primeng/button'
import { MessageModule } from 'primeng/message'

export default {
    title: 'App/HostTab',
    component: HostTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([
                    { path: 'dhcp/hosts/:id', component: HostTabComponent },
                    { path: 'iframe.html', component: HostTabComponent },
                ]),
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                ChipModule,
                ConfirmDialogModule,
                FieldsetModule,
                FormsModule,
                MessageModule,
                PopoverModule,
                ProgressSpinnerModule,
                TableModule,
                RouterModule,
                ToastModule,
                ToggleButtonModule,
                TreeModule,
                TagModule,
            ],
            declarations: [
                HostDataSourceLabelComponent,
                IdentifierComponent,
                DhcpClientClassSetViewComponent,
                DhcpOptionSetViewComponent,
                HelpTipComponent,
                EntityLinkComponent,
                ByteCharacterComponent,
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
