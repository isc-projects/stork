import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { GlobalSearchComponent } from './global-search.component'
import { SearchService } from '../backend/api/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { PopoverModule } from 'primeng/popover'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormsModule } from '@angular/forms'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter, RouterModule } from '@angular/router'

describe('GlobalSearchComponent', () => {
    let component: GlobalSearchComponent
    let fixture: ComponentFixture<GlobalSearchComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [GlobalSearchComponent],
            imports: [PopoverModule, NoopAnimationsModule, FormsModule, RouterModule],
            providers: [
                SearchService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
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

    it('should display app name and proper link to an app in the results', async () => {
        component.searchResults = {
            subnets: { items: [] },
            sharedNetworks: { items: [] },
            hosts: { items: [] },
            machines: { items: [] },
            apps: { items: [{ type: 'kea', id: 1, name: 'dhcp-server' }] },
            users: { items: [] },
            groups: { items: [] },
        }

        // Show search result box, by default it is hidden
        component.searchResultsBox.show(new Event('click'), fixture.nativeElement)
        await fixture.whenRenderingDone()
        fixture.detectChanges()

        const appsDiv = fixture.debugElement.query(By.css('#apps-div'))
        expect(appsDiv).toBeDefined()
        expect(appsDiv.children.length).toBe(2)
        const appDiv = appsDiv.children[1]
        expect(appDiv.children.length).toBe(1)
        const appAnchor = appDiv.children[0]
        expect(appAnchor.nativeElement.innerText).toBe('dhcp-server')
        expect(appAnchor.attributes.href).toBe('/apps/1')
    })
})
