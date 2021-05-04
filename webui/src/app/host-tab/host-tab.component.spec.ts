import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

import { FieldsetModule } from 'primeng/fieldset'
import { TableModule } from 'primeng/table'

import { of } from 'rxjs'

import { DHCPService } from '../backend'
import { HostTabComponent } from './host-tab.component'

describe('HostTabComponent', () => {
    let component: HostTabComponent
    let fixture: ComponentFixture<HostTabComponent>
    let dhcpApi: DHCPService

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService],
            imports: [FieldsetModule, FormsModule, HttpClientTestingModule, NoopAnimationsModule, TableModule],
            declarations: [HostTabComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostTabComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display host information', () => {
        const host = {
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
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
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

        const addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::1')
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::2')

        const prefixReservationsFieldset = fixture.debugElement.query(By.css('#prefix-reservations-fieldset'))
        expect(prefixReservationsFieldset).toBeTruthy()
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:2::/64')
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:3::/64')

        const nonIPReservationsFieldset = fixture.debugElement.query(By.css('#non-ip-reservations-fieldset'))
        expect(nonIPReservationsFieldset).toBeTruthy()
        expect(nonIPReservationsFieldset.nativeElement.textContent).toContain('mouse.example.org')

        const hostIdsFieldset = fixture.debugElement.query(By.css('#dhcp-identifiers-fieldset'))
        expect(hostIdsFieldset).toBeTruthy()
        expect(hostIdsFieldset.nativeElement.textContent).toContain('duid')
        expect(hostIdsFieldset.nativeElement.textContent).toContain('hw-address')
        expect(hostIdsFieldset.nativeElement.textContent).toContain('01:02:03:04')
        expect(hostIdsFieldset.nativeElement.textContent).toContain('f1:f2:f3:f4:f5:f6')

        const appsFieldset = fixture.debugElement.query(By.css('#apps-fieldset'))
        expect(appsFieldset).toBeTruthy()

        const appLinks = appsFieldset.queryAll(By.css('a'))
        expect(appLinks.length).toBe(2)
        expect(appLinks[0].properties.routerLink).toBe('/apps/kea/1')
        expect(appLinks[1].properties.routerLink).toBe('/apps/kea/2')

        let configTag = appsFieldset.query(By.css('.cfg-srctag'))
        expect(configTag).toBeTruthy()
        expect(configTag.nativeElement.innerText).toBe('config')
        configTag = appsFieldset.query(By.css('.hostcmds-srctag'))
        expect(configTag).toBeTruthy()
        expect(configTag.nativeElement.innerText).toBe('host_cmds')
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

    it('should display lease information', () => {
        const host = {
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
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
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

        const addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        const addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        let addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(2)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in use')
        expect(addressReservationTrs[1].nativeElement.textContent).toContain('unused')

        const expandAddressLink = addressReservationTrs[0].query(By.css('a'))
        expect(expandAddressLink).toBeTruthy()
        expandAddressLink.nativeElement.click()
        fixture.detectChanges()

        addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(3)
        expect(addressReservationTrs[1].nativeElement.textContent).toContain(
            'Found 1 assigned lease with the expiration time at '
        )

        const prefixReservationsFieldset = fixture.debugElement.query(By.css('#prefix-reservations-fieldset'))
        expect(prefixReservationsFieldset).toBeTruthy()
        const prefixReservationTable = prefixReservationsFieldset.query(By.css('table'))
        expect(prefixReservationTable).toBeTruthy()
        const prefixReservationTrs = prefixReservationTable.queryAll(By.css('tr'))
        expect(prefixReservationTrs.length).toBe(2)
        expect(prefixReservationTrs[0].nativeElement.textContent).toContain('in conflict')
        expect(prefixReservationTrs[1].nativeElement.textContent).toContain('unused')
    })

    it('should display multiple lease information', () => {
        const host = {
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
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
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

        let addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        let addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        let addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(1)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in use')

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

        addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(1)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in conflict')
    })

    it('should return correct lease summary', () => {
        // Single lease in use.
        let leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
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
            },
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 assigned leases with the latest expiration time at')

        // Single expired lease.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 lease for this reservation which expired at')

        // Two expired leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 leases for this reservation. They include a lease which expired at')

        // Single declined lease.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 lease for this reservation which is declined and has expiration time at')

        // Two declined leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain(
            'Found 2 leases for this reservation. They include a declined lease with expiration time at'
        )

        // Single conflicted lease with MAC address.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found a lease with expiration time at')
        expect(summary).toContain(
            'assigned to the client with MAC address=1a:1b:1c:1d:1e:1f, for which it was not reserved.'
        )

        // Conflicted lease with DUID.
        const leaseInfo2 = {
            culprit: {
                duid: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    duid: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo2)
        expect(summary).toContain('Found a lease with expiration time at')
        expect(summary).toContain('assigned to the client with DUID=11:12:13, for which it was not reserved.')

        // Conflicted lease with client-id.
        const leaseInfo3 = {
            culprit: {
                clientId: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    clientId: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo3)
        expect(summary).toContain('Found a lease with expiration time at')
        expect(summary).toContain('assigned to the client with client-id=11:12:13, for which it was not reserved.')
    })
})
