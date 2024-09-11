import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { SharedNetworkTabComponent } from './shared-network-tab.component'
import { FieldsetModule } from 'primeng/fieldset'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { DividerModule } from 'primeng/divider'
import { ChartModule } from 'primeng/chart'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { TooltipModule } from 'primeng/tooltip'
import { TreeModule } from 'primeng/tree'
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { IPType } from '../iptype'
import { By } from '@angular/platform-browser'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { RouterModule } from '@angular/router'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'
import { DHCPService } from '../backend'
import { HttpErrorResponse } from '@angular/common/http'
import { ParameterViewComponent } from '../parameter-view/parameter-view.component'
import { UnhyphenPipe } from '../pipes/unhyphen.pipe'
import { UncamelPipe } from '../pipes/uncamel.pipe'
import { PositivePipe } from '../pipes/positive.pipe'

describe('SharedNetworkTabComponent', () => {
    let component: SharedNetworkTabComponent
    let fixture: ComponentFixture<SharedNetworkTabComponent>
    let dhcpApi: DHCPService
    let msgService: MessageService
    let confirmService: ConfirmationService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                AddressPoolBarComponent,
                CascadedParametersBoardComponent,
                DhcpOptionSetViewComponent,
                DelegatedPrefixBarComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                LocalNumberPipe,
                ParameterViewComponent,
                PlaceholderPipe,
                PositivePipe,
                UncamelPipe,
                UnhyphenPipe,
                SharedNetworkTabComponent,
                SubnetBarComponent,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
            ],
            imports: [
                ButtonModule,
                ChartModule,
                CheckboxModule,
                ConfirmDialogModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                RouterModule.forRoot([{ path: 'dhcp/shared-networks/:id', component: SharedNetworkTabComponent }]),
                TableModule,
                TagModule,
                TooltipModule,
                TreeModule,
            ],
            providers: [ConfirmationService, MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedNetworkTabComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        confirmService = fixture.debugElement.injector.get(ConfirmationService)
        msgService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display an IPv4 shared network', () => {
        component.sharedNetwork = {
            name: 'foo',
            addrUtilization: 30,
            pools: [
                {
                    pool: '192.0.2.1-192.0.2.10',
                },
                {
                    pool: '192.0.2.100-192.0.2.110',
                },
                {
                    pool: '192.0.2.150-192.0.2.160',
                },
                {
                    pool: '192.0.3.1-192.0.3.10',
                },
                {
                    pool: '192.0.3.100-192.0.3.110',
                },
                {
                    pool: '192.0.3.150-192.0.3.160',
                },
            ],
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                },
                {
                    id: 2,
                    subnet: '192.0.3.0/24',
                },
            ],
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            clientClass: 'zab',
                            requireClientClasses: ['bar'],
                            ddnsGeneratedPrefix: 'herhost',
                            ddnsOverrideClientUpdate: false,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'foo.example.org',
                            ddnsReplaceClientName: 'always',
                            ddnsSendUpdates: false,
                            ddnsUpdateOnRenew: true,
                            ddnsUseConflictResolution: false,
                            fourOverSixInterface: 'nn',
                            fourOverSixInterfaceID: 'ofo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1700,
                            minPreferredLifetime: 1500,
                            maxPreferredLifetime: 1900,
                            reservationMode: 'in-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: false,
                            renewTimer: 1900,
                            rebindTimer: 2500,
                            t1Percent: 0.26,
                            t2Percent: 0.74,
                            calculateTeeTimes: true,
                            validLifetime: 3700,
                            minValidLifetime: 3500,
                            maxValidLifetime: 4000,
                            allocator: 'flq',
                            authoritative: true,
                            bootFileName: '/tmp/boot.1',
                            _interface: 'eth1',
                            interfaceID: 'foo',
                            matchClientID: true,
                            nextServer: '192.1.2.4',
                            options: [
                                {
                                    code: 5,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['8.8.8.8'],
                                        },
                                    ],
                                    universe: IPType.IPv4,
                                },
                            ],
                            optionsHash: '234',
                            pdAllocator: 'iterative',
                            rapidCommit: false,
                            relay: {
                                ipAddresses: ['192.0.2.2'],
                            },
                            serverHostname: 'off.example.org',
                            storeExtendedInfo: false,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            clientClass: 'abc',
                            requireClientClasses: [],
                            ddnsGeneratedPrefix: 'hishost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: false,
                            ddnsQualifyingSuffix: 'uff.example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'enp0s8',
                            fourOverSixInterfaceID: 'idx',
                            fourOverSixSubnet: '2001:db8:1:1::/64',
                            hostnameCharReplacement: 'Y',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1600,
                            minPreferredLifetime: 1400,
                            maxPreferredLifetime: 1800,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: true,
                            reservationsInSubnet: false,
                            reservationsOutOfPool: true,
                            renewTimer: 1800,
                            rebindTimer: 2400,
                            t1Percent: 0.24,
                            t2Percent: 0.7,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3900,
                            allocator: 'iterative',
                            authoritative: false,
                            bootFileName: '/tmp/bootx',
                            _interface: 'eth0',
                            interfaceID: 'uffa',
                            matchClientID: false,
                            nextServer: '10.1.1.1',
                            options: [
                                {
                                    code: 23,
                                    fields: [
                                        {
                                            fieldType: 'uint8',
                                            values: ['10'],
                                        },
                                    ],
                                    universe: IPType.IPv4,
                                },
                            ],
                            optionsHash: '345',
                            pdAllocator: 'random',
                            rapidCommit: true,
                            serverHostname: 'abc.example.org',
                            storeExtendedInfo: false,
                        },
                    },
                },
                {
                    appId: 2,
                    appName: 'foo@192.0.2.2',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            clientClass: 'zab',
                            requireClientClasses: ['bar'],
                            ddnsGeneratedPrefix: 'herhost',
                            ddnsOverrideClientUpdate: false,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'foo.example.org',
                            ddnsReplaceClientName: 'always',
                            ddnsSendUpdates: false,
                            ddnsUpdateOnRenew: true,
                            ddnsUseConflictResolution: false,
                            fourOverSixInterface: 'nn',
                            fourOverSixInterfaceID: 'ofo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1700,
                            minPreferredLifetime: 1500,
                            maxPreferredLifetime: 1900,
                            reservationMode: 'in-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: false,
                            renewTimer: 1900,
                            rebindTimer: 2500,
                            t1Percent: 0.26,
                            t2Percent: 0.74,
                            calculateTeeTimes: true,
                            validLifetime: 3700,
                            minValidLifetime: 3500,
                            maxValidLifetime: 4000,
                            allocator: 'flq',
                            authoritative: true,
                            bootFileName: '/tmp/boot.1',
                            _interface: 'eth1',
                            interfaceID: 'foo',
                            matchClientID: true,
                            nextServer: '192.1.2.4',
                            pdAllocator: 'iterative',
                            rapidCommit: false,
                            relay: {
                                ipAddresses: ['192.0.2.2'],
                            },
                            serverHostname: 'off.example.org',
                            storeExtendedInfo: false,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            clientClass: 'abc',
                            requireClientClasses: [],
                            ddnsGeneratedPrefix: 'hishost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: false,
                            ddnsQualifyingSuffix: 'uff.example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'enp0s8',
                            fourOverSixInterfaceID: 'idx',
                            fourOverSixSubnet: '2001:db8:1:1::/64',
                            hostnameCharReplacement: 'Y',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1600,
                            minPreferredLifetime: 1400,
                            maxPreferredLifetime: 1800,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: true,
                            reservationsInSubnet: false,
                            reservationsOutOfPool: true,
                            renewTimer: 1800,
                            rebindTimer: 2400,
                            t1Percent: 0.24,
                            t2Percent: 0.7,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3900,
                            allocator: 'iterative',
                            authoritative: false,
                            bootFileName: '/tmp/bootx',
                            _interface: 'eth0',
                            interfaceID: 'uffa',
                            matchClientID: false,
                            nextServer: '10.1.1.1',
                            pdAllocator: 'random',
                            rapidCommit: true,
                            serverHostname: 'abc.example.org',
                            storeExtendedInfo: false,
                        },
                    },
                },
            ],
        }

        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Shared Network foo')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Shared Network')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.2')

        expect(fieldsets[1].nativeElement.innerText).toContain('Subnets')
        const subnetBars = fieldsets[1].queryAll(By.css('app-subnet-bar'))
        expect(subnetBars.length).toBe(2)
        expect(subnetBars[0].nativeElement.innerText).toContain('192.0.2.0/24')
        expect(subnetBars[1].nativeElement.innerText).toContain('192.0.3.0/24')

        expect(fieldsets[2].nativeElement.innerText).toContain('Pools')

        const poolBars = fieldsets[2].queryAll(By.css('app-address-pool-bar'))
        expect(poolBars.length).toBe(6)
        expect(poolBars[0].nativeElement.innerText).toContain('192.0.2.1-192.0.2.10')
        expect(poolBars[1].nativeElement.innerText).toContain('192.0.2.100-192.0.2.110')
        expect(poolBars[2].nativeElement.innerText).toContain('192.0.2.150-192.0.2.160')
        expect(poolBars[3].nativeElement.innerText).toContain('192.0.3.1-192.0.3.10')
        expect(poolBars[4].nativeElement.innerText).toContain('192.0.3.100-192.0.3.110')
        expect(poolBars[5].nativeElement.innerText).toContain('192.0.3.150-192.0.3.160')

        const charts = fieldsets[3].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(1)

        expect(fieldsets[4].nativeElement.innerText).toContain('Cache Threshold')
        expect(fieldsets[4].nativeElement.innerText).toContain('0.3')
        expect(fieldsets[4].nativeElement.innerText).toContain('900')

        // Ensure that the DHCP options are excluded from this list.
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options')
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options Hash')

        // DHCP options sit in their own fieldset.
        expect(fieldsets[5].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[5].nativeElement.innerText).toContain('8.8.8.8')
    })

    it('should display an IPv4 shared network with minimal data', () => {
        component.sharedNetwork = {
            name: 'bar',
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                },
            ],
        }

        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Shared Network bar')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(5)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Shared Network')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')

        expect(fieldsets[1].nativeElement.innerText).toContain('Subnets')
        expect(fieldsets[1].nativeElement.innerText).toContain('No subnets configured.')

        expect(fieldsets[2].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[2].nativeElement.innerText).toContain('No pools configured.')

        expect(fieldsets[3].nativeElement.innerText).toContain('DHCP Parameters')
        expect(fieldsets[3].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[4].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display an IPv6 shared network', () => {
        component.sharedNetwork = {
            name: 'foo',
            universe: IPType.IPv6,
            addrUtilization: 30,
            pdUtilization: 60,
            pools: [
                {
                    pool: '2001:db8:1::2-2001:db8:1::786',
                },
                {
                    pool: '2001:db8:2::2-2001:db8:2::786',
                },
            ],
            subnets: [
                {
                    id: 1,
                    subnet: '2001:db8:1::/64',
                },
                {
                    id: 2,
                    subnet: '2001:db8:2::/64',
                },
            ],
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                        },
                    },
                },
            ],
        }

        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Shared Network foo')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Shared Network')
        expect(fieldsets[0].nativeElement.innerText).toContain('foo@192.0.2.1')

        expect(fieldsets[1].nativeElement.innerText).toContain('Subnets')
        const subnetBars = fieldsets[1].queryAll(By.css('app-subnet-bar'))
        expect(subnetBars.length).toBe(2)
        expect(subnetBars[0].nativeElement.innerText).toContain('2001:db8:1::/64')
        expect(subnetBars[1].nativeElement.innerText).toContain('2001:db8:2::/64')

        expect(fieldsets[2].nativeElement.innerText).toContain('Pools')

        const poolBars = fieldsets[2].queryAll(By.css('app-address-pool-bar'))
        expect(poolBars.length).toBe(2)
        expect(poolBars[0].nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::786')
        expect(poolBars[1].nativeElement.innerText).toContain('2001:db8:2::2-2001:db8:2::786')

        const charts = fieldsets[3].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(2)

        expect(fieldsets[4].nativeElement.innerText).toContain('Hostname Char Replacement')
        expect(fieldsets[4].nativeElement.innerText).toContain('X')
        expect(fieldsets[4].nativeElement.innerText).toContain('[^A-Za-z0-9.-]')

        // Ensure that the DHCP options are excluded from this list.
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options')
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options Hash')

        // DHCP options sit in their own fieldset.
        expect(fieldsets[5].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[5].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display shared network delete button', () => {
        component.sharedNetwork = {
            name: 'foo',
            universe: IPType.IPv6,
            addrUtilization: 30,
            pdUtilization: 60,
            pools: [
                {
                    pool: '2001:db8:1::2-2001:db8:1::786',
                },
            ],
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {},
                    },
                },
            ],
        }
        fixture.detectChanges()
        const deleteBtn = fixture.debugElement.query(By.css('[label=Delete]'))
        expect(deleteBtn).toBeTruthy()

        // Simulate clicking on the button and make sure that the confirm dialog
        // has been displayed.
        spyOn(confirmService, 'confirm')
        deleteBtn.nativeElement.click()
        expect(confirmService.confirm).toHaveBeenCalled()
    })

    it('should emit an event indicating successful shared network deletion', fakeAsync(() => {
        const successResp: any = {}
        spyOn(dhcpApi, 'deleteSharedNetwork').and.returnValue(of(successResp))
        spyOn(msgService, 'add')
        spyOn(component.sharedNetworkDelete, 'emit')

        // Delete the subnet.
        component.sharedNetwork = {
            id: 1,
        }
        component.deleteSharedNetwork()
        tick()
        // Success message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // An event should be called.
        expect(component.sharedNetworkDelete.emit).toHaveBeenCalledWith(component.sharedNetwork)
        // This flag should be cleared.
        expect(component.sharedNetworkDeleting).toBeFalse()
    }))

    it('should not emit an event when shared network deletion fails', fakeAsync(() => {
        spyOn(dhcpApi, 'deleteSharedNetwork').and.returnValue(throwError(() => new HttpErrorResponse({ status: 404 })))
        spyOn(msgService, 'add')
        spyOn(component.sharedNetworkDelete, 'emit')

        // Delete the host and receive an error.
        component.sharedNetwork = {
            id: 1,
        }
        component.deleteSharedNetwork()
        tick()
        // Error message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // The event shouldn't be emitted on error.
        expect(component.sharedNetworkDelete.emit).not.toHaveBeenCalledWith(component.sharedNetwork)
        // This flag should be cleared.
        expect(component.sharedNetworkDeleting).toBeFalse()
    }))
})
