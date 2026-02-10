import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { KeaGlobalConfigurationPageComponent } from './kea-global-configuration-page.component'
import { provideRouter } from '@angular/router'
import { of, throwError } from 'rxjs'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { KeaDaemonConfig, ServicesService } from '../backend'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { HttpErrorResponse, HttpEvent, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { DaemonsPageComponent } from '../daemons-page/daemons-page.component'
import { RouterTestingHarness } from '@angular/router/testing'

describe('KeaGlobalConfigurationPageComponent', () => {
    let component: KeaGlobalConfigurationPageComponent
    let fixture: ComponentFixture<any>
    let messageService: MessageService
    let servicesService: ServicesService
    let validDaemonConfig: KeaDaemonConfig

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([
                    {
                        path: 'daemons/:id',
                        component: DaemonsPageComponent,
                    },
                    {
                        path: 'daemons/:daemonId/global-config',
                        component: KeaGlobalConfigurationPageComponent,
                    },
                ]),
            ],
        }).compileComponents()

        const harness = await RouterTestingHarness.create()
        component = await harness.navigateByUrl('/daemons/1/global-config', KeaGlobalConfigurationPageComponent)
        fixture = harness.fixture

        messageService = fixture.debugElement.injector.get(MessageService)
        servicesService = fixture.debugElement.injector.get(ServicesService)

        fixture.detectChanges()

        validDaemonConfig = {
            daemonId: 1,
            daemonName: 'dhcp4',
            daemonLabel: 'DHCPv4@localhost',
            editable: true,
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
        }
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should exclude parameters', () => {
        expect(component.excludedParameters.length).toBe(11)

        expect(component.excludedParameters).toContain('clientClasses')
        expect(component.excludedParameters).toContain('configControl')
        expect(component.excludedParameters).toContain('hooksLibraries')
        expect(component.excludedParameters).toContain('loggers')
        expect(component.excludedParameters).toContain('optionData')
        expect(component.excludedParameters).toContain('optionDef')
        expect(component.excludedParameters).toContain('optionsHash')
        expect(component.excludedParameters).toContain('reservations')
        expect(component.excludedParameters).toContain('sharedNetworks')
        expect(component.excludedParameters).toContain('subnet4')
        expect(component.excludedParameters).toContain('subnet6')
    })

    it('should fetch global configuration parameters', fakeAsync(() => {
        spyOn(servicesService, 'getDaemonConfig').and.returnValue(of(validDaemonConfig) as any)

        component.daemonId = 1
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesService.getDaemonConfig).toHaveBeenCalledOnceWith(1)

        expect(component.loaded).toBeTrue()
        expect(component.disableEdit).toBeFalse()

        expect(component.dhcpParameters.length).toBe(1)
        expect(component.dhcpParameters.at(0).name).toBe('DHCPv4@localhost')
        expect(component.dhcpParameters.at(0).parameters.length).toBe(1)
        expect(component.dhcpParameters.at(0).parameters[0].hasOwnProperty('allocator'))
        expect(component.dhcpParameters.at(0).parameters[0]['allocator']).toBe('iterative')

        const parametersBoard = fixture.debugElement.query(By.css('app-cascaded-parameters-board'))
        expect(parametersBoard).toBeTruthy()
    }))

    it('should disable edit button for global parameters', fakeAsync(() => {
        const config: KeaDaemonConfig = {
            daemonId: 1,
            daemonName: 'dhcp4',
            daemonLabel: 'DHCPv4@localhost',
            editable: false,
            config: {
                Dhcp4: {
                    allocator: 'iterative',
                },
            },
        }
        spyOn(servicesService, 'getDaemonConfig').and.returnValue(of(config as HttpEvent<KeaDaemonConfig>))

        component.daemonId = 1
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesService.getDaemonConfig).toHaveBeenCalledOnceWith(1)

        expect(component.loaded).toBeTrue()
        expect(component.disableEdit).toBeTrue()
    }))

    it('should update breadcrumbs before and after fetching global configuration parameters', fakeAsync(() => {
        spyOn(component, 'updateBreadcrumbs')
        spyOn(servicesService, 'getDaemonConfig').and.returnValue(of(validDaemonConfig as any))

        component.ngOnInit()
        expect(component.updateBreadcrumbs).toHaveBeenCalledOnceWith(1)

        tick()
        fixture.detectChanges()

        expect(component.updateBreadcrumbs).toHaveBeenCalledTimes(2)
        expect(component.updateBreadcrumbs).toHaveBeenCalledWith(1, 'DHCPv4@localhost')
    }))

    it('should display a message on error', fakeAsync(() => {
        spyOn(messageService, 'add')
        spyOn(servicesService, 'getDaemonConfig').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )

        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalledTimes(1)
        expect(component.disableEdit).toBeTrue()
    }))

    it('should unsubscribe on destroy', () => {
        spyOn(component.subscriptions, 'unsubscribe')
        component.ngOnDestroy()
        expect(component.subscriptions.unsubscribe).toHaveBeenCalledTimes(1)
    })

    it('should update breadcrumbs if only daemon is provided', () => {
        component.updateBreadcrumbs(1)
        fixture.detectChanges()

        const breadcrumbs = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
            .componentInstance as BreadcrumbsComponent
        expect(breadcrumbs).toBeTruthy()

        expect(breadcrumbs.items.length).toBe(4)
        expect(breadcrumbs.items[2].label).toBe('Daemon')
        expect(breadcrumbs.items[2].routerLink).toBe('/daemons/1')
    })

    it('should update breadcrumbs if daemon id and daemon name are provided', () => {
        component.updateBreadcrumbs(1, 'My Daemon')
        fixture.detectChanges()

        const breadcrumbs = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
            .componentInstance as BreadcrumbsComponent
        expect(breadcrumbs).toBeTruthy()

        expect(breadcrumbs.items.length).toBe(4)
        expect(breadcrumbs.items[2].label).toBe('My Daemon')
        expect(breadcrumbs.items[2].routerLink).toBe('/daemons/1')
    })
})
