import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

import { MessageService } from 'primeng/api'

import { LeaseSearchPageComponent } from './lease-search-page.component'
import { DHCPService } from '../backend'

describe('LeaseSearchPageComponent', () => {
    let component: LeaseSearchPageComponent
    let fixture: ComponentFixture<LeaseSearchPageComponent>
    let dhcpApi: DHCPService

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, MessageService],
            imports: [FormsModule, HttpClientTestingModule],
            declarations: [LeaseSearchPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LeaseSearchPageComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should ignore empty search text', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        spyOn(dhcpApi, 'getLeases')

        // Simulate typing only spaces in the search box.
        searchInputElement.value = '    '
        searchInputElement.dispatchEvent(new Event('input'))
        searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
        fixture.detectChanges()

        // Make sure that search is not triggered.
        expect(dhcpApi.getLeases).not.toHaveBeenCalled()
    })

    it('should trigger leases search', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        const fakeLeases: any = {
            data: {
                items: [],
                total: 0,
                erredApps: [],
            },
        }
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))

        // Simulate typing only spaces in the search box.
        searchInputElement.value = '192.1.0.1'
        searchInputElement.dispatchEvent(new Event('input'))
        searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
        fixture.detectChanges()

        expect(component.searchText).toBe('192.1.0.1')

        // Make sure that search is triggered.
        expect(dhcpApi.getLeases).toHaveBeenCalled()
    })

    it('should return correct lease state name', () => {
        expect(component.leaseStateAsText(null)).toBe('Valid')
        expect(component.leaseStateAsText(0)).toBe('Valid')
        expect(component.leaseStateAsText(1)).toBe('Declined')
        expect(component.leaseStateAsText(2)).toBe('Expired/Reclaimed')
        expect(component.leaseStateAsText(3)).toBe('Unknown')
    })
})
