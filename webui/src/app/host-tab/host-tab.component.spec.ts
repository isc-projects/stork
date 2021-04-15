import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

import { FieldsetModule } from 'primeng/fieldset'
import { TableModule } from 'primeng/table'

import { HostTabComponent } from './host-tab.component'

describe('HostTabComponent', () => {
    let component: HostTabComponent
    let fixture: ComponentFixture<HostTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [FieldsetModule, NoopAnimationsModule, TableModule],
            declarations: [HostTabComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostTabComponent)
        component = fixture.componentInstance
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
                    address: '2001:db8:2::',
                },
                {
                    address: '2001:db8:3::',
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
        component.host = host
        fixture.detectChanges()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[1] Host in subnet 2001:db8:1::/64')

        const addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::1')
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::2')

        const prefixReservationsFieldset = fixture.debugElement.query(By.css('#prefix-reservations-fieldset'))
        expect(prefixReservationsFieldset).toBeTruthy()
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:2::')
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:3::')

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
})
