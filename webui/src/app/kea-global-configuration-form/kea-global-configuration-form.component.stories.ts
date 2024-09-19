import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { ActivatedRoute, convertToParamMap } from '@angular/router'
import { MockParamMap } from '../utils'
import { of } from 'rxjs'
import { MessageService } from 'primeng/api'
import { importProvidersFrom } from '@angular/core'
import { HttpClientModule } from '@angular/common/http'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { FieldsetModule } from 'primeng/fieldset'
import { TableModule } from 'primeng/table'
import { ButtonModule } from 'primeng/button'
import { KeaGlobalConfigurationFormComponent } from './kea-global-configuration-form.component'
import { UpdateKeaDaemonsGlobalParametersBeginResponse } from '../backend'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DropdownModule } from 'primeng/dropdown'
import { InputNumberModule } from 'primeng/inputnumber'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { MessagesModule } from 'primeng/messages'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { MultiSelectModule } from 'primeng/multiselect'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { SplitButtonModule } from 'primeng/splitbutton'

const mockUpdateKeaGlobalParameters4BeginData: UpdateKeaDaemonsGlobalParametersBeginResponse = {
    id: 123,
    configs: [
        {
            daemonId: 1,
            daemonName: 'dhcp4',
            appId: 1,
            appName: 'kea@agent1',
            appType: 'kea',
            config: {
                Dhcp4: {
                    allocator: 'iterative',
                    authoritative: false,
                    'boot-file-name': '',
                    'calculate-tee-times': false,
                    'client-classes': [
                        {
                            'boot-file-name': '',
                            name: 'class-00-00',
                            'next-server': '0.0.0.0',
                            'option-data': [],
                            'option-def': [],
                            'server-hostname': '',
                            test: "substring(hexstring(pkt4.mac,':'),0,5) == '00:00'",
                        },
                    ],
                    'config-control': {
                        'config-databases': [
                            {
                                host: 'mariadb',
                                name: 'agent_kea',
                                password: 'agent_kea',
                                type: 'mysql',
                                user: 'agent_kea',
                            },
                        ],
                        'config-fetch-wait-time': 20,
                    },
                    'control-socket': {
                        'socket-name': '/tmp/kea4-ctrl-socket',
                        'socket-type': 'unix',
                    },
                    'ddns-conflict-resolution-mode': 'check-with-dhcid',
                    'ddns-generated-prefix': 'myhost',
                    'ddns-override-client-update': false,
                    'ddns-override-no-update': false,
                    'ddns-qualifying-suffix': '',
                    'ddns-replace-client-name': 'never',
                    'ddns-send-updates': true,
                    'ddns-update-on-renew': false,
                    'decline-probation-period': 86400,
                    'dhcp-ddns': {
                        'enable-updates': false,
                        'max-queue-size': 1024,
                        'ncr-format': 'JSON',
                        'ncr-protocol': 'UDP',
                        'sender-ip': '0.0.0.0',
                        'sender-port': 0,
                        'server-ip': '127.0.0.1',
                        'server-port': 53001,
                    },
                    'dhcp-queue-control': {
                        capacity: 64,
                        'enable-queue': false,
                        'queue-type': 'kea-ring4',
                    },
                    'dhcp4o6-port': 0,
                    'early-global-reservations-lookup': false,
                    'echo-client-id': true,
                    'expired-leases-processing': {
                        'flush-reclaimed-timer-wait-time': 25,
                        'hold-reclaimed-time': 3600,
                        'max-reclaim-leases': 100,
                        'max-reclaim-time': 250,
                        'reclaim-timer-wait-time': 10,
                        'unwarned-reclaim-cycles': 5,
                    },
                    'hooks-libraries': [
                        {
                            library: '/usr/lib/x86_64-linux-gnu/kea/hooks/libdhcp_lease_cmds.so',
                        },
                    ],
                    'host-reservation-identifiers': ['hw-address', 'duid', 'circuit-id', 'client-id'],
                    'hostname-char-replacement': '',
                    'hostname-char-set': '[^A-Za-z0-9.-]',
                    'interfaces-config': {
                        interfaces: ['*'],
                        're-detect': true,
                    },
                    'ip-reservations-unique': true,
                    'lease-database': {
                        host: 'mariadb',
                        name: 'agent_kea',
                        password: 'agent_kea',
                        type: 'mysql',
                        user: 'agent_kea',
                    },
                    loggers: [
                        {
                            debuglevel: 0,
                            name: 'kea-dhcp4',
                            'output-options': [
                                {
                                    flush: true,
                                    output: 'stdout',
                                    pattern: '%-5p %m\n',
                                },
                            ],
                            severity: 'DEBUG',
                        },
                    ],
                    'match-client-id': true,
                    'multi-threading': {
                        'enable-multi-threading': true,
                        'packet-queue-size': 64,
                        'thread-pool-size': 0,
                    },
                    'next-server': '0.0.0.0',
                    'option-data': [
                        {
                            'always-send': false,
                            code: 6,
                            'csv-format': true,
                            data: '192.0.2.1, 192.0.2.2',
                            name: 'domain-name-servers',
                            'never-send': false,
                            space: 'dhcp4',
                        },
                    ],
                    'option-def': [],
                    'parked-packet-limit': 256,
                    'rebind-timer': 120,
                    'renew-timer': 90,
                    reservations: [
                        {
                            'boot-file-name': '',
                            'client-classes': [],
                            'client-id': 'AAAAAAAAAAAA',
                            hostname: '',
                            'ip-address': '10.0.0.222',
                            'next-server': '0.0.0.0',
                            'option-data': [],
                            'server-hostname': '',
                        },
                    ],
                    'reservations-global': false,
                    'reservations-in-subnet': true,
                    'reservations-lookup-first': false,
                    'reservations-out-of-pool': false,
                    'sanity-checks': {
                        'extended-info-checks': 'fix',
                        'lease-checks': 'warn',
                    },
                    'server-hostname': '',
                    'server-tag': '',
                },
            },
        },
    ],
}

export default {
    title: 'App/KeaGlobalConfigurationForm',
    component: KeaGlobalConfigurationFormComponent,
    argTypes: {
        formGroup: {
            table: {
                disable: true,
            },
        },
    },
    decorators: [
        applicationConfig({
            providers: [
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(convertToParamMap({ daemonId: '1' })),
                    },
                },
                MessageService,
                importProvidersFrom(HttpClientModule),
                importProvidersFrom(NoopAnimationsModule),
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                InputNumberModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
                TableModule,
                TagModule,
                TriStateCheckboxModule,
                ToastModule,
                SplitButtonModule,
            ],
            declarations: [
                ArrayValueSetFormComponent,
                SharedParametersFormComponent,
                DhcpOptionSetFormComponent,
                DhcpOptionFormComponent,
            ],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<KeaGlobalConfigurationFormComponent>

export const UpdateConfiguration: Story = {
    args: {
        daemonId: 1,
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/kea-global-parameters/transaction',
                method: 'POST',
                status: 200,
                delay: 2000,
                response: mockUpdateKeaGlobalParameters4BeginData,
            },
        ],
    },
}

export const ErrorMessage: Story = {
    args: {
        daemonId: 1,
    },
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/kea-global-parameters/transaction',
                method: 'POST',
                status: 500,
                delay: 1000,
                response: mockUpdateKeaGlobalParameters4BeginData,
            },
        ],
    },
}
