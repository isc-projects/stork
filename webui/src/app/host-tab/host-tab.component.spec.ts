import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

import { FieldsetModule } from 'primeng/fieldset'
import { ConfirmationService, MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { ConfirmDialogModule } from 'primeng/confirmdialog'

import { of, throwError } from 'rxjs'

import { DHCPService, Host, Lease } from '../backend'
import { HostTabComponent } from './host-tab.component'
import { RouterModule } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { IdentifierComponent } from '../identifier/identifier.component'
import { TreeModule } from 'primeng/tree'
import { DhcpClientClassSetViewComponent } from '../dhcp-client-class-set-view/dhcp-client-class-set-view.component'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TagModule } from 'primeng/tag'
import { ChipModule } from 'primeng/chip'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { DividerModule } from 'primeng/divider'
import { HostDataSourceLabelComponent } from '../host-data-source-label/host-data-source-label.component'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { MessagesModule } from 'primeng/messages'

describe('HostTabComponent', () => {
    let component: HostTabComponent
    let fixture: ComponentFixture<HostTabComponent>
    let dhcpApi: DHCPService
    let msgService: MessageService
    let confirmService: ConfirmationService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, ConfirmationService, MessageService],
            imports: [
                ConfirmDialogModule,
                ChipModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                TableModule,
                RouterModule,
                RouterTestingModule,
                ToggleButtonModule,
                TreeModule,
                TagModule,
                MessagesModule,
                ProgressSpinnerModule,
            ],
            declarations: [
                DhcpClientClassSetViewComponent,
                DhcpOptionSetViewComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HostTabComponent,
                IdentifierComponent,
                HostDataSourceLabelComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostTabComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        confirmService = fixture.debugElement.injector.get(ConfirmationService)
        msgService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display v4 host information', () => {
        const host: Partial<Host> = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '51:52:53:54:55:56',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.123',
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
                    nextServer: '192.0.2.2',
                    serverHostname: 'my-server',
                    bootFileName: '/tmp/bootfile1',
                    hostname: 'mouse.example.org',
                    ipReservations: [
                        {
                            address: '192.0.2.123',
                        },
                    ],
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    nextServer: '192.0.2.2',
                    serverHostname: 'my-server',
                    bootFileName: '/tmp/bootfile1',
                    hostname: 'mouse.example.org',
                    ipReservations: [
                        {
                            address: '192.0.2.123',
                        },
                    ],
                },
            ],
        }
        const fakeLeases: any = {}
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[1] Host in subnet 192.0.2.0/24')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)

        const ipReservationsFieldset = fieldsets[3]
        expect(ipReservationsFieldset).toBeTruthy()
        expect(ipReservationsFieldset.nativeElement.textContent).toContain('192.0.2.123')

        const hostnameFieldset = fieldsets[2]
        expect(hostnameFieldset).toBeTruthy()
        expect(hostnameFieldset.nativeElement.textContent).toContain('mouse.example.org')

        const hostIdsFieldset = fieldsets[1]
        expect(hostIdsFieldset).toBeTruthy()
        expect(hostIdsFieldset.nativeElement.textContent).toContain('hw-address')
        // HW address should remain in hexadecimal form.
        expect(hostIdsFieldset.nativeElement.textContent).toContain('51:52:53:54:55:56')

        const appsFieldset = fieldsets[0]
        expect(appsFieldset).toBeTruthy()

        const appLinks = appsFieldset.queryAll(By.css('a'))
        expect(appLinks.length).toBe(2)
        expect(appLinks[0].attributes.href).toBe('/apps/kea/1')
        expect(appLinks[1].attributes.href).toBe('/apps/kea/2')

        let datasourceLabel = appsFieldset.query(By.css('.datasource--config'))
        expect(datasourceLabel).toBeTruthy()
        expect(datasourceLabel.nativeElement.innerText).toBe('config')
        datasourceLabel = appsFieldset.query(By.css('.datasource--hostcmds'))
        expect(datasourceLabel).toBeTruthy()
        expect(datasourceLabel.nativeElement.innerText).toBe('host_cmds')

        const bootFieldsFieldset = fieldsets[4]
        expect(bootFieldsFieldset).toBeTruthy()
        expect(bootFieldsFieldset.nativeElement.textContent).toContain('192.0.2.2')
        expect(bootFieldsFieldset.nativeElement.textContent).toContain('my-server')
        expect(bootFieldsFieldset.nativeElement.textContent).toContain('/tmp/bootfile1')
    })

    it('should display v6 host information', () => {
        const host: Partial<Host> = {
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
                    hostname: 'mouse.example.org',
                    ipReservations: [
                        {
                            address: '2001:db8:1::1',
                        },
                        {
                            address: '2001:db8:1::2',
                        },
                        {
                            address: '2001:db8:2::/64',
                        },
                        {
                            address: '2001:db8:3::/64',
                        },
                    ],
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    hostname: 'mouse.example.org',
                    ipReservations: [
                        {
                            address: '2001:db8:1::1',
                        },
                        {
                            address: '2001:db8:1::2',
                        },
                        {
                            address: '2001:db8:2::/64',
                        },
                        {
                            address: '2001:db8:3::/64',
                        },
                    ],
                },
            ],
        }
        const fakeLeases: any = {}
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[1] Host in subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        const ipReservationsFieldset = fieldsets[3]
        expect(ipReservationsFieldset).toBeTruthy()
        expect(ipReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::1')
        expect(ipReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::2')
        expect(ipReservationsFieldset.nativeElement.textContent).toContain('2001:db8:2::/64')
        expect(ipReservationsFieldset.nativeElement.textContent).toContain('2001:db8:3::/64')

        const hostnameFieldset = fieldsets[2]
        expect(hostnameFieldset).toBeTruthy()
        expect(hostnameFieldset.nativeElement.textContent).toContain('mouse.example.org')

        const hostIdsFieldset = fieldsets[1]
        expect(hostIdsFieldset).toBeTruthy()
        expect(hostIdsFieldset.nativeElement.textContent).toContain('duid')
        expect(hostIdsFieldset.nativeElement.textContent).toContain('hw-address')
        // DUID should be converted to textual form.
        expect(hostIdsFieldset.nativeElement.textContent).toContain('QRST')
        // HW address should remain in hexadecimal form.
        expect(hostIdsFieldset.nativeElement.textContent).toContain('51:52:53:54:55:56')

        const appsFieldset = fieldsets[0]
        expect(appsFieldset).toBeTruthy()

        const appLinks = appsFieldset.queryAll(By.css('a'))
        expect(appLinks.length).toBe(2)
        expect(appLinks[0].attributes.href).toBe('/apps/kea/1')
        expect(appLinks[1].attributes.href).toBe('/apps/kea/2')

        let datasourceLabel = appsFieldset.query(By.css('.datasource--config'))
        expect(datasourceLabel).toBeTruthy()
        expect(datasourceLabel.nativeElement.innerText).toBe('config')
        datasourceLabel = appsFieldset.query(By.css('.datasource--hostcmds'))
        expect(datasourceLabel).toBeTruthy()
        expect(datasourceLabel.nativeElement.innerText).toBe('host_cmds')
    })

    it('should display global host tab title', () => {
        const host = {
            id: 2,
            subnetId: 0,
        }
        component.host = host
        fixture.detectChanges()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[2] Global host')
    })

    it('should handle error while fetching host information', () => {
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')
        const host = {
            id: 1,
        }
        component.host = host
        expect(dhcpApi.getLeases).toHaveBeenCalled()
        expect(msgService.add).toHaveBeenCalled()
    })

    it('should display lease information', () => {
        const host: Partial<Host> = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
                {
                    idType: 'hw-address',
                    idHexValue: 'f1:f2:f3:f4:f5:f6',
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
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                    ipReservations: [
                        {
                            address: '2001:db8:1::1',
                        },
                        {
                            address: '2001:db8:1::2',
                        },
                        {
                            address: '2001:db8:2::/64',
                        },
                        {
                            address: '2001:db8:3::/64',
                        },
                    ],
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    ipReservations: [
                        {
                            address: '2001:db8:1::1',
                        },
                        {
                            address: '2001:db8:1::2',
                        },
                        {
                            address: '2001:db8:2::/64',
                        },
                        {
                            address: '2001:db8:3::/64',
                        },
                    ],
                },
            ],
        }
        const fakeLeases: any = {
            items: [
                {
                    id: 1,
                    ipAddress: '2001:db8:1::1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
                {
                    id: 2,
                    ipAddress: '2001:db8:2::',
                    prefixLength: 64,
                    state: 0,
                    duid: 'e1:e2:e3:e4:e5:e6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
            ],
            conflicts: [2],
            erredApps: [],
        }
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        const ipReservationsFieldset = fieldsets[3]
        expect(ipReservationsFieldset).toBeTruthy()
        const ipReservationTable = ipReservationsFieldset.query(By.css('table'))
        expect(ipReservationTable).toBeTruthy()
        let ipReservationTrs = ipReservationTable.queryAll(By.css('tr'))
        expect(ipReservationTrs.length).toBe(4)
        expect(ipReservationTrs[0].nativeElement.textContent).toContain('in use')
        expect(ipReservationTrs[1].nativeElement.textContent).toContain('unused')
        expect(ipReservationTrs[2].nativeElement.textContent).toContain('in conflict')
        expect(ipReservationTrs[3].nativeElement.textContent).toContain('unused')

        let links = ipReservationTrs[0].queryAll(By.css('a'))
        expect(links.length).toBe(1)
        expect(links[0].attributes.href).toBe('/dhcp/leases?text=2001:db8:1::1')
        expect(links[0].properties.text).toBe('2001:db8:1::1')

        links = ipReservationTrs[2].queryAll(By.css('a'))
        expect(links[0].attributes.href).toBe('/dhcp/leases?text=2001:db8:2::')
        expect(links[0].properties.text).toBe('2001:db8:2::/64')

        const expandAddressLink = ipReservationTrs[0].query(By.css('button'))
        expect(expandAddressLink).toBeTruthy()
        expandAddressLink.nativeElement.click()
        fixture.detectChanges()

        ipReservationTrs = ipReservationTable.queryAll(By.css('tr'))
        expect(ipReservationTrs.length).toBe(5)
        expect(ipReservationTrs[1].nativeElement.textContent).toContain(
            'Found 1 assigned lease with the expiration time at '
        )
    })

    it('should display multiple lease information', () => {
        const host: Partial<Host> = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: 'f1:f2:f3:f4:f5:f6',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                    ipReservations: [
                        {
                            address: '192.0.2.1',
                        },
                    ],
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                    ipReservations: [
                        {
                            address: '192.0.2.1',
                        },
                    ],
                },
            ],
        }

        const fakeLeases: any = {
            items: [
                {
                    id: 1,
                    ipAddress: '192.0.2.1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
                {
                    id: 2,
                    ipAddress: '192.0.2.1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
            ],
            conflicts: [],
            erredApps: [],
        }
        const spy = spyOn(dhcpApi, 'getLeases')

        spy.and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        let ipReservationsFieldset = fieldsets[3]
        expect(ipReservationsFieldset).toBeTruthy()
        let ipReservationTable = ipReservationsFieldset.query(By.css('table'))
        expect(ipReservationTable).toBeTruthy()
        let ipReservationTrs = ipReservationTable.queryAll(By.css('tr'))
        expect(ipReservationTrs.length).toBe(1)
        expect(ipReservationTrs[0].nativeElement.textContent).toContain('in use')

        // Simulate the case that conflicted lease is returned. Note that here
        // we also simulate different order of leases.
        fakeLeases.items[1] = fakeLeases.items[0]
        fakeLeases.items[0] = {
            id: 2,
            ipAddress: '192.0.2.1',
            state: 0,
            hwAddress: 'e1:e2:e3:e4:e5:e6',
            subnetId: 1,
            cltt: 1616149050,
            validLifetime: 3600,
        }
        fakeLeases.conflicts.push(2)
        spy.and.returnValue(of(fakeLeases))
        component.refreshLeases()
        expect(dhcpApi.getLeases).toHaveBeenCalled()
        fixture.detectChanges()

        fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        ipReservationsFieldset = fieldsets[3]
        expect(ipReservationsFieldset).toBeTruthy()
        ipReservationTable = ipReservationsFieldset.query(By.css('table'))
        expect(ipReservationTable).toBeTruthy()
        ipReservationTrs = ipReservationTable.queryAll(By.css('tr'))
        expect(ipReservationTrs.length).toBe(1)
        expect(ipReservationTrs[0].nativeElement.textContent).toContain('in conflict')
    })

    it('should return correct lease summary', () => {
        // Single lease in use.
        let leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        let summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 assigned lease with the expiration time at')

        // Two leases in use.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 assigned leases with the latest expiration time at')

        // Single expired lease.
        // Set cltt so that the expiration time elapses 10 or more seconds ago.
        const testCltt = new Date().getTime() / 1000 - 3610
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: testCltt,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: testCltt,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toMatch(
            /Found 1 lease for this reservation that expired at \d{4}-\d{2}-\d{2}\s\d{2}\:\d{2}\:\d{2} \(\d{2} s ago\)/
        )

        // Two expired leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 leases for this reservation. This includes a lease that expired at')

        // Single declined lease.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 lease for this reservation which is declined and has an expiration time at')

        // Two declined leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain(
            'Found 2 leases for this reservation. This includes a declined lease with expiration time at'
        )

        // Single conflicted lease with MAC address.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Conflicted,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain(
            'assigned to the client with MAC address=1a:1b:1c:1d:1e:1f, for which it was not reserved.'
        )

        // Conflicted lease with DUID.
        const leaseInfo2 = {
            culprit: {
                duid: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Conflicted,
            leases: [
                {
                    duid: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo2)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain('assigned to the client with DUID=11:12:13, for which it was not reserved.')

        // Conflicted lease with client-id.
        const leaseInfo3 = {
            culprit: {
                clientId: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            } as Lease,
            usage: component.Usage.Conflicted,
            leases: [
                {
                    clientId: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                } as Lease,
            ],
        }
        summary = component.getLeaseSummary(leaseInfo3)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain('assigned to the client with client-id=11:12:13, for which it was not reserved.')
    })

    it('should display host delete button for host reservation received over host_cmds', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()
        const deleteBtn = fixture.debugElement.query(By.css('[label=Delete]'))
        expect(deleteBtn).toBeTruthy()

        // Simulate clicking on the button and make sure that the confirm dialog
        // has been displayed.
        spyOn(confirmService, 'confirm')
        deleteBtn.nativeElement.click()
        expect(confirmService.confirm).toHaveBeenCalled()
    })

    it('should emit an event indicating successful host deletion', fakeAsync(() => {
        const successResp: any = {}
        spyOn(dhcpApi, 'deleteHost').and.returnValue(of(successResp))
        spyOn(msgService, 'add')
        spyOn(component.hostDelete, 'emit')

        // Delete the host.
        component.host = {
            id: 1,
        }
        component.deleteHost()
        tick()
        // Success message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // An event should be called.
        expect(component.hostDelete.emit).toHaveBeenCalledWith(component.host)
        // This flag should be cleared.
        expect(component.hostDeleted).toBeFalse()
    }))

    it('should not emit an event when host deletion fails', fakeAsync(() => {
        spyOn(dhcpApi, 'deleteHost').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')
        spyOn(component.hostDelete, 'emit')

        // Delete the host and receive an error.
        component.host = {
            id: 1,
        }
        component.deleteHost()
        tick()
        // Error message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // The event shouldn't be emitted on error.
        expect(component.hostDelete.emit).not.toHaveBeenCalledWith(component.host)
        // This flag should be cleared.
        expect(component.hostDeleted).toBeFalse()
    }))

    it('should not display host delete button for host reservation from the config file', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()
        // Unable to delete hosts specified in the config file.
        expect(fixture.debugElement.query(By.css('[label=Delete]'))).toBeFalsy()
    })

    it('should display different host data for different servers separately', () => {
        const host: Partial<Host> = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
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
                    address: '2001:db8:2::/80',
                },
                {
                    address: '2001:db8:3::/80',
                },
            ],
            hostname: 'foo.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    clientClasses: ['foo', 'bar'],
                    nextServer: '192.0.2.1',
                    serverHostname: 'myhostname',
                    bootFileName: '/tmp/boot1',
                    hostname: 'foo.example.org',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                    ipReservations: [
                        {
                            address: '2001:db8:1::1',
                        },
                        {
                            address: '2001:db8:1::2',
                        },
                        {
                            address: '2001:db8:2::/80',
                        },
                        {
                            address: '2001:db8:3::/80',
                        },
                    ],
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                    clientClasses: ['baz'],
                    nextServer: '192.0.2.2',
                    serverHostname: 'yourhostname',
                    bootFileName: '/tmp/boot2',
                    hostname: 'bar.example.org',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1026,
                        },
                    ],
                    optionsHash: '2222',
                    ipReservations: [
                        {
                            address: '2001:db8:1::3',
                        },
                        {
                            address: '2001:db8:4::/80',
                        },
                    ],
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(12)

        expect(fieldsets[2].properties.innerText).toContain('Hostname')
        expect(fieldsets[3].properties.innerText).toContain('Hostname')
        expect(fieldsets[4].properties.innerText).toContain('IP Reservations')
        expect(fieldsets[5].properties.innerText).toContain('IP Reservations')
        expect(fieldsets[6].properties.innerText).toContain('Boot Fields')
        expect(fieldsets[7].properties.innerText).toContain('Boot Fields')
        expect(fieldsets[8].properties.innerText).toContain('Client Classes')
        expect(fieldsets[9].properties.innerText).toContain('Client Classes')
        expect(fieldsets[10].properties.innerText).toContain('DHCP Options')
        expect(fieldsets[11].properties.innerText).toContain('DHCP Options')

        for (let i = 2; i < 12; i++) {
            let link = fieldsets[i].query(By.css('a'))
            expect(link).toBeTruthy()
            if (i % 2 === 0) {
                expect(link.properties.innerText).toContain('frog')
                expect(link.properties.pathname).toBe('/apps/kea/1')
            } else {
                expect(link.properties.innerText).toContain('lion')
                expect(link.properties.pathname).toBe('/apps/kea/2')
            }
        }
    })

    it('should display the same host data for different servers in one panel', () => {
        const host: Partial<Host> = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            prefixReservations: [
                {
                    address: '192.0.0.128/30',
                },
            ],
            hostname: 'foo.example.com',
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    nextServer: '192.0.2.2',
                    serverHostname: 'my-server',
                    bootFileName: '/tmp/boot1',
                    clientClasses: ['foo', 'bar'],
                    hostname: 'foo.example.com',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                    ipReservations: [
                        {
                            address: '192.0.2.1',
                        },
                        {
                            address: '192.0.0.128/30',
                        },
                    ],
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                    nextServer: '192.0.2.2',
                    serverHostname: 'my-server',
                    bootFileName: '/tmp/boot1',
                    clientClasses: ['foo', 'bar'],
                    hostname: 'foo.example.com',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                    ipReservations: [
                        {
                            address: '192.0.2.1',
                        },
                        {
                            address: '192.0.0.128/30',
                        },
                    ],
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)

        expect(fieldsets[4].properties.innerText).toContain('Boot Fields')
        expect(fieldsets[4].properties.innerText).toContain('All Servers')

        expect(fieldsets[5].properties.innerText).toContain('Client Classes')
        expect(fieldsets[5].properties.innerText).toContain('All Servers')

        expect(fieldsets[6].properties.innerText).toContain('DHCP Options')
        expect(fieldsets[6].properties.innerText).toContain('All Servers')
    })

    it('should display DHCP options panel for host with one daemon', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    nextServer: '192.0.2.1',
                    clientClasses: ['foo', 'bar'],
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)

        expect(fieldsets[4].properties.innerText).toContain('Boot Fields')
        expect(fieldsets[4].properties.innerText).toContain('All Servers')
        expect(fieldsets[5].properties.innerText).toContain('Client Classes')
        expect(fieldsets[5].properties.innerText).toContain('All Servers')
        expect(fieldsets[6].properties.innerText).toContain('DHCP Options')
        expect(fieldsets[6].properties.innerText).toContain('All Servers')
    })

    it('should display a message about no client classes configured', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)
        expect(fieldsets[4].properties.innerText).toContain('No client classes configured.')
    })

    it('should display dashes when boot fields are not specified', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '192.0.2.0::/24',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    nextServer: '0.0.0.0',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(7)
        expect(fieldsets[4].properties.innerText).toContain('Next server\n0.0.0.0')
        expect(fieldsets[4].properties.innerText).toContain('Server hostname\n-')
        expect(fieldsets[4].properties.innerText).toContain('Boot file name\n-')
    })

    it('should display a message about no DHCP options configured', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)
        expect(fieldsets[5].properties.innerText).toContain('No options configured.')
    })

    it('should group local hosts by app ID properly', () => {
        const host = {
            localHosts: [
                {
                    appId: 3,
                    daemonId: 31,
                },
                {
                    appId: 3,
                    daemonId: 32,
                },
                {
                    appId: 3,
                    daemonId: 33,
                },
                {
                    appId: 2,
                    daemonId: 21,
                },
                {
                    appId: 2,
                    daemonId: 22,
                },
                {
                    appId: 1,
                    daemonId: 11,
                },
            ],
        } as Host

        component.host = host
        const groups = component.localHostsGroups.appID

        expect(groups.length).toBe(3)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId
            expect(group.length).toBe(appId)
            for (let item of group) {
                expect(item.daemonId).toBeGreaterThan(10 * appId)
                expect(item.daemonId).toBeLessThan(10 * (appId + 1))
            }
        }
    })

    it('should group local hosts by different boot fields properly', () => {
        const host = {
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    bootFileName: 'foo',
                    serverHostname: 'bar',
                    nextServer: 'baz',
                },
                {
                    appId: 1,
                    daemonId: 2,
                    bootFileName: 'foo',
                    serverHostname: 'bar',
                    nextServer: 'baz',
                },
                {
                    appId: 1,
                    daemonId: 3,
                    bootFileName: 'oof',
                    serverHostname: 'rab',
                    nextServer: 'zab',
                },
                {
                    appId: 2,
                    daemonId: 4,
                    bootFileName: 'foo',
                    serverHostname: 'bar',
                    nextServer: 'baz',
                },
                {
                    appId: 2,
                    daemonId: 5,
                    bootFileName: 'foo',
                    serverHostname: 'bar',
                    nextServer: 'baz',
                },
            ],
        } as Host

        component.host = host
        const groups = component.localHostsGroups.bootFields

        expect(groups.length).toBe(4)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId

            switch (appId) {
                case 1:
                    expect(group.length).toBe(1)
                    expect(group[0].daemonId).toBeLessThanOrEqual(3)
                    break
                case 2:
                    expect(group.length).toBe(2)
                    for (const item of group) {
                        expect(item.daemonId).toBeGreaterThanOrEqual(4)
                    }
            }
        }
    })

    it('should group local hosts by different client classes properly', () => {
        const host = {
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    clientClasses: ['foo', 'bar'],
                },
                {
                    appId: 1,
                    daemonId: 2,
                    clientClasses: ['foo', 'bar'],
                },
                {
                    appId: 1,
                    daemonId: 3,
                    clientClasses: ['oof', 'rab'],
                },
                {
                    appId: 2,
                    daemonId: 4,
                    clientClasses: ['foo', 'bar'],
                },
                {
                    appId: 2,
                    daemonId: 5,
                    clientClasses: ['foo', 'bar'],
                },
            ],
        } as Host

        component.host = host
        const groups = component.localHostsGroups.clientClasses

        expect(groups.length).toBe(4)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId

            switch (appId) {
                case 1:
                    expect(group.length).toBe(1)
                    expect(group[0].daemonId).toBeLessThanOrEqual(3)
                    break
                case 2:
                    expect(group.length).toBe(2)
                    for (const item of group) {
                        expect(item.daemonId).toBeGreaterThanOrEqual(4)
                    }
            }
        }
    })

    it('should group local hosts by different hash of DHCP options properly', () => {
        const host: Partial<Host> = {
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    optionsHash: 'foo',
                },
                {
                    appId: 1,
                    daemonId: 2,
                    optionsHash: 'foo',
                },
                {
                    appId: 1,
                    daemonId: 3,
                    optionsHash: 'oof',
                },
                {
                    appId: 2,
                    daemonId: 4,
                    optionsHash: 'foo',
                },
                {
                    appId: 2,
                    daemonId: 5,
                    optionsHash: 'foo',
                },
            ],
        }

        component.host = host
        const groups = component.localHostsGroups.dhcpOptions

        expect(groups.length).toBe(4)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId

            switch (appId) {
                case 1:
                    expect(group.length).toBe(1)
                    expect(group[0].daemonId).toBeLessThanOrEqual(3)
                    break
                case 2:
                    expect(group.length).toBe(2)
                    for (const item of group) {
                        expect(item.daemonId).toBeGreaterThanOrEqual(4)
                    }
            }
        }
    })

    it('should group local hosts by different IP reservations', () => {
        const host: Partial<Host> = {
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    ipReservations: [
                        {
                            address: '10.0.0.1',
                        },
                        {
                            address: '10.0.0.2',
                        },
                    ],
                },
                {
                    appId: 1,
                    daemonId: 2,
                    ipReservations: [
                        {
                            address: '10.0.0.1',
                        },
                        {
                            address: '10.0.0.2',
                        },
                    ],
                },
                {
                    appId: 2,
                    daemonId: 3,
                    ipReservations: [
                        {
                            address: '10.0.0.3',
                        },
                    ],
                },
                {
                    appId: 2,
                    daemonId: 4,
                    ipReservations: [
                        {
                            address: '10.0.0.3',
                        },
                        {
                            address: '10.0.0.4',
                        },
                    ],
                },
            ],
        }

        component.host = host
        const groups = component.localHostsGroups.ipReservations

        expect(groups.length).toBe(3)

        for (let group of groups) {
            const appId = group[0].appId

            switch (appId) {
                case 1:
                    expect(group.length).toBe(2)
                    expect(group[0].daemonId).toBeLessThanOrEqual(2)
                    break
                case 2:
                    expect(group.length).toBe(1)
                    expect(group[0].daemonId).toBeGreaterThanOrEqual(3)
                    expect(group[0].daemonId).toBeLessThanOrEqual(4)
            }
        }
    })

    it('should group local hosts by different hostnames', () => {
        const host: Partial<Host> = {
            hostname: 'ignored',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    hostname: 'foo',
                },
                {
                    appId: 1,
                    daemonId: 2,
                    hostname: 'foo',
                },
                {
                    appId: 2,
                    daemonId: 3,
                    hostname: 'bar',
                },
                {
                    appId: 2,
                    daemonId: 4,
                },
            ],
        }

        component.host = host
        const groups = component.localHostsGroups.hostname

        expect(groups.length).toBe(3)

        for (let group of groups) {
            const appId = group[0].appId

            switch (appId) {
                case 1:
                    expect(group.length).toBe(2)
                    expect(group[0].daemonId).toBeLessThanOrEqual(2)
                    break
                case 2:
                    expect(group.length).toBe(1)
                    expect(group[0].daemonId).toBeGreaterThanOrEqual(3)
                    expect(group[0].daemonId).toBeLessThanOrEqual(4)
            }
        }
    })

    it('should group all local hosts into a single group if there are no differences', () => {
        const host = {
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                },
                {
                    appId: 1,
                    daemonId: 2,
                },
                {
                    appId: 1,
                    daemonId: 3,
                },
                {
                    appId: 2,
                    daemonId: 4,
                },
                {
                    appId: 2,
                    daemonId: 5,
                },
            ],
        } as Host

        component.host = host

        for (const groups of [
            component.localHostsGroups.bootFields,
            component.localHostsGroups.clientClasses,
            component.localHostsGroups.dhcpOptions,
            component.localHostsGroups.hostname,
            component.localHostsGroups.ipReservations,
        ]) {
            expect(groups.length).toBe(1)
            const group = groups[0]
            expect(group.length).toBe(5)
        }
    })
})
