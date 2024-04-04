import { HostFormComponent } from './host-form.component'

import { StoryObj, Meta, moduleMetadata } from '@storybook/angular'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormsModule, ReactiveFormsModule, UntypedFormBuilder } from '@angular/forms'
import { HttpClientModule } from '@angular/common/http'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { InputNumberModule } from 'primeng/inputnumber'
import { InputSwitchModule } from 'primeng/inputswitch'
import { MessagesModule } from 'primeng/messages'
import { MultiSelectModule } from 'primeng/multiselect'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { SplitButtonModule } from 'primeng/splitbutton'
import { TableModule } from 'primeng/table'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { ToastModule } from 'primeng/toast'
import { toastDecorator } from '../utils-stories'
import { CreateHostBeginResponse, DHCPService, UpdateHostBeginResponse } from '../backend'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'

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
        moduleMetadata({
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientModule,
                InputNumberModule,
                InputSwitchModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                RouterTestingModule,
                SplitButtonModule,
                TableModule,
                ToastModule,
                ToggleButtonModule,
            ],
            declarations: [
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                HelpTipComponent,
                HostFormComponent,
            ],
            providers: [UntypedFormBuilder, DHCPService, MessageService],
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
