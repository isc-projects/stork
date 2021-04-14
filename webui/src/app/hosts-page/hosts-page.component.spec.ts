import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { DHCPService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { By } from '@angular/platform-browser'
import { of, BehaviorSubject } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>
    let router: Router
    let route: ActivatedRoute
    let paramMap: any
    let paramMapSubject: BehaviorSubject<any>
    let paramMapSpy: any

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService],
            imports: [
                FormsModule,
                TableModule,
                HttpClientTestingModule,
                RouterTestingModule.withRoutes([
                    {
                        path: 'dhcp/hosts',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/hosts/all',
                    },
                    {
                        path: 'dhcp/hosts/:id',
                        component: HostsPageComponent,
                    },
                ]),
            ],
            declarations: [EntityLinkComponent, HostsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        router = fixture.debugElement.injector.get(Router)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        paramMap = convertToParamMap({})
        paramMapSubject = new BehaviorSubject(paramMap)
        paramMapSpy = spyOnProperty(route, 'paramMap').and.returnValue(paramMapSubject)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTab).toBe(component.tabs[0])
        expect(component.filterText.length).toBe(0)
    })

    it('host table should have valid app name and app link', () => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()
        // Table rows have ids created by appending host id to the host-row- string.
        const row = fixture.debugElement.query(By.css('#host-row-1'))
        // There should be 6 table cells in the row.
        expect(row.children.length).toBe(6)
        // The last one includes the app name.
        const appNameTd = row.children[5]
        // The cell includes a link to the app.
        expect(appNameTd.children.length).toBe(1)
        const appLink = appNameTd.children[0]
        expect(appLink.nativeElement.innerText).toBe('frog config')
        // Verify that the link to the app is correct.
        expect(appLink.properties.hasOwnProperty('pathname')).toBeTrue()
        expect(appLink.properties.pathname).toBe('/apps/kea/1')
    })

    it('should open and close host tabs', () => {
        // Create a list with two hosts.
        component.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '11:12:13:14',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Open tab with host with id 1.
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTab).toBe(component.tabs[1])

        // Open tab with host with id 2.
        paramMapSubject.next(convertToParamMap({ id: 2 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTab).toBe(component.tabs[2])

        // Navigate back to the hosts list in the first tab.
        paramMapSubject.next(convertToParamMap({}))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTab).toBe(component.tabs[0])

        // Navigate to the existing tab with host with id 1.
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTab).toBe(component.tabs[1])

        // Close the middle tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTab).toBe(component.tabs[0])

        // Close the right tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTab).toBe(component.tabs[0])
    })

    it('should generate a label from host information', () => {
        const host0 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host0)).toBe('192.0.2.1')

        const host1 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host1)).toBe('2001:db8::')

        const host2 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            hostname: 'mouse.example.org',
        }
        expect(component.getHostLabel(host2)).toBe('mouse.example.org')

        const host3 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
        }
        expect(component.getHostLabel(host3)).toBe('duid=01:02:03:04')

        const host4 = {
            id: 1,
        }
        expect(component.getHostLabel(host4)).toBe('[1]')
    })
})
