import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { DHCPService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(convertToParamMap({ id: 1 })),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [FormsModule, TableModule, HttpClientTestingModule],
            declarations: [HostsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
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
        console.info(appLink.nativeElement)
        // Verify that the link to the app is correct.
        expect(appLink.properties.hasOwnProperty('routerLink')).toBeTrue()
        expect(appLink.properties.routerLink).toBe('/apps/kea/1')
    })
})
