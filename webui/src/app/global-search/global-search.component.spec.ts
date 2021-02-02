import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { GlobalSearchComponent } from './global-search.component'
import { SearchService } from '../backend/api/api'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'

describe('GlobalSearchComponent', () => {
    let component: GlobalSearchComponent
    let fixture: ComponentFixture<GlobalSearchComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [GlobalSearchComponent],
            providers: [SearchService],
            imports: [HttpClientTestingModule],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(GlobalSearchComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display app name and proper link to an app in the results', () => {
        component.searchResults = {
            subnets: { items: [] },
            sharedNetworks: { items: [] },
            hosts: { items: [] },
            machines: { items: [] },
            apps: { items: [{ type: 'kea', id: 1, name: 'dhcp-server' }] },
            users: { items: [] },
            groups: { items: [] },
        }
        fixture.detectChanges()
        const appsDiv = fixture.debugElement.query(By.css('#apps-div'))
        expect(appsDiv.children.length).toBe(2)
        const appDiv = appsDiv.children[1]
        expect(appDiv.children.length).toBe(1)
        const appAnchor = appDiv.children[0]
        expect(appAnchor.nativeElement.innerText).toBe('dhcp-server')
        expect(appAnchor.properties.hasOwnProperty('routerLink')).toBeTrue()
        expect(appAnchor.properties.routerLink).toBe('/apps/kea/1')
    })
})
