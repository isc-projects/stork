import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { KeaGlobalConfigurationFormComponent } from './kea-global-configuration-form.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import {
    DHCPService,
    UpdateKeaDaemonsGlobalParametersBeginResponse,
    UpdateKeaDaemonsGlobalParametersSubmitRequest,
} from '../backend'
import { of, throwError } from 'rxjs'
import { FieldsetModule } from 'primeng/fieldset'
import { MessagesModule } from 'primeng/messages'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { ReactiveFormsModule } from '@angular/forms'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { InputNumberModule } from 'primeng/inputnumber'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { ChipsModule } from 'primeng/chips'
import { MultiSelectModule } from 'primeng/multiselect'
import { By } from '@angular/platform-browser'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { SplitButtonModule } from 'primeng/splitbutton'

describe('KeaGlobalConfigurationFormComponent', () => {
    let component: KeaGlobalConfigurationFormComponent
    let fixture: ComponentFixture<KeaGlobalConfigurationFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService

    let cannedResponseBegin4: UpdateKeaDaemonsGlobalParametersBeginResponse = {
        id: 123,
        configs: [
            {
                daemonId: 1,
                daemonName: 'dhcp4',
                appId: 1,
                appName: 'kea@agent1',
                appType: 'kea',
                daemonVersion: '3.0.0',
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
                        'ddns-use-conflict-resolution': true,
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
                options: {
                    optionsHash: 'hash',
                    options: [
                        {
                            alwaysSend: false,
                            code: 6,
                            fields: [
                                {
                                    fieldType: 'ipv4-address',
                                    values: ['192.0.2.1'],
                                },
                                {
                                    fieldType: 'ipv4-address',
                                    values: ['192.0.2.2'],
                                },
                            ],
                            universe: 4,
                        },
                    ],
                },
            },
        ],
    }

    let cannedResponseBegin6: UpdateKeaDaemonsGlobalParametersBeginResponse = {
        id: 234,
        configs: [
            {
                daemonId: 2,
                daemonName: 'dhcp6',
                appId: 1,
                appName: 'kea@agent1',
                appType: 'kea',
                config: {
                    Dhcp6: {
                        allocator: 'random',
                        'pd-allocator': 'flq',
                    },
                },
            },
        ],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FieldsetModule,
                HttpClientTestingModule,
                InputNumberModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
                SplitButtonModule,
            ],
            declarations: [
                ArrayValueSetFormComponent,
                KeaGlobalConfigurationFormComponent,
                SharedParametersFormComponent,
                DhcpOptionSetFormComponent,
                DhcpOptionFormComponent,
            ],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(KeaGlobalConfigurationFormComponent)
        component = fixture.componentInstance
        component.daemonId = 1
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should open a form for updating global Kea DHCPv4 configuration', fakeAsync(() => {
        spyOn(dhcpApi, 'updateKeaGlobalParametersBegin').and.returnValue(of(cannedResponseBegin4 as any))
        component.daemonId = 1
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.response?.id).toBe(123)
        expect(component.formGroup).toBeTruthy()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateKeaGlobalParametersSubmit').and.returnValue(of(okResp))
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const request: UpdateKeaDaemonsGlobalParametersSubmitRequest = {
            configs: [
                {
                    daemonId: 1,
                    daemonName: 'dhcp4',
                    partialConfig: {
                        allocator: 'iterative',
                        authoritative: false,
                        ddnsConflictResolutionMode: 'check-with-dhcid',
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: false,
                        ddnsOverrideNoUpdate: false,
                        ddnsQualifyingSuffix: '',
                        ddnsReplaceClientName: 'never',
                        ddnsSendUpdates: true,
                        ddnsUpdateOnRenew: false,
                        dhcpDdnsEnableUpdates: false,
                        dhcpDdnsMaxQueueSize: 1024,
                        dhcpDdnsNcrFormat: 'JSON',
                        dhcpDdnsNcrProtocol: 'UDP',
                        dhcpDdnsSenderIP: '0.0.0.0',
                        dhcpDdnsSenderPort: 0,
                        dhcpDdnsServerIP: '127.0.0.1',
                        dhcpDdnsServerPort: 53001,
                        earlyGlobalReservationsLookup: false,
                        echoClientId: true,
                        expiredFlushReclaimedTimerWaitTime: 25,
                        expiredHoldReclaimedTime: 3600,
                        expiredMaxReclaimLeases: 100,
                        expiredMaxReclaimTime: 250,
                        expiredReclaimTimerWaitTime: 10,
                        expiredUnwarnedReclaimCycles: 5,
                        hostReservationIdentifiers: ['hw-address', 'duid', 'circuit-id', 'client-id'],
                        reservationsGlobal: false,
                        reservationsInSubnet: true,
                        reservationsOutOfPool: false,
                        options: [
                            {
                                alwaysSend: false,
                                code: 6,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.1'],
                                    },
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.2'],
                                    },
                                ],
                                options: [],
                                universe: 4,
                            },
                        ],
                    },
                },
            ],
        }

        expect(dhcpApi.updateKeaGlobalParametersSubmit).toHaveBeenCalledWith(component.response?.id, request)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a form for updating global Kea DHCPv6 configuration', fakeAsync(() => {
        spyOn(dhcpApi, 'updateKeaGlobalParametersBegin').and.returnValue(of(cannedResponseBegin6 as any))
        component.daemonId = 2
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.response?.id).toBe(234)
        expect(component.formGroup).toBeTruthy()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateKeaGlobalParametersSubmit').and.returnValue(of(okResp))
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const request: UpdateKeaDaemonsGlobalParametersSubmitRequest = {
            configs: [
                {
                    daemonId: 2,
                    daemonName: 'dhcp6',
                    partialConfig: {
                        allocator: 'random',
                        pdAllocator: 'flq',
                        options: [],
                    },
                },
            ],
        }

        expect(dhcpApi.updateKeaGlobalParametersSubmit).toHaveBeenCalledWith(component.response?.id, request)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should emit cancel event', () => {
        spyOn(component.formCancel, 'emit')
        component.onCancel()
        expect(component.formCancel.emit).toHaveBeenCalled()
    })

    it('should present an error message when begin transaction fails', fakeAsync(() => {
        spyOn(dhcpApi, 'updateKeaGlobalParametersBegin').and.returnValues(
            throwError(() => new Error('status: 404')),
            of(cannedResponseBegin4 as any)
        )
        component.daemonId = 1
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.initError).toEqual('status: 404')

        const messagesElement = fixture.debugElement.query(By.css('p-messages'))
        expect(messagesElement).toBeTruthy()
        expect(messagesElement.nativeElement.outerText).toContain(component.initError)

        const retryButton = fixture.debugElement.query(By.css('[label="Retry"]'))
        expect(retryButton).toBeTruthy()
        expect(retryButton.nativeElement.outerText).toBe('Retry')

        component.onRetry()
        tick()
        fixture.detectChanges()
        tick()

        expect(fixture.debugElement.query(By.css('p-messages'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Retry"]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Submit"]'))).toBeTruthy()
    }))

    it('should list the server names', () => {
        const response = {
            id: 1,
            configs: [
                {
                    appName: 'foo',
                    daemonName: 'dhcp4',
                    config: { Dhcp4: {} },
                },
                {
                    appName: 'bar',
                    daemonName: 'dhcp4',
                    config: { Dhcp4: {} },
                },
            ],
        } as UpdateKeaDaemonsGlobalParametersBeginResponse
        component.response = response

        expect(component.servers.length).toBe(2)
        expect(component.servers).toContain('foo/dhcp4')
        expect(component.servers).toContain('bar/dhcp4')
    })

    it('should detect IPv6 servers', () => {
        component.response = { configs: [{ daemonName: 'dhcp4' }] }
        expect(component.isIPv6).toBeFalse()

        component.response = { configs: [{ daemonName: 'dhcp6' }] }
        expect(component.isIPv6).toBeTrue()

        component.response = { configs: [{ daemonName: 'dhcp4' }, { daemonName: 'dhcp6' }] }
        expect(component.isIPv6).toBeFalse()

        component.response = { configs: [{ daemonName: 'dhcp6' }, { daemonName: 'dhcp4' }] }
        expect(component.isIPv6).toBeTrue()

        component.response = null
        expect(component.isIPv6).toBeFalse()

        component.response = { configs: [] }
        expect(component.isIPv6).toBeFalse()
    })
})
