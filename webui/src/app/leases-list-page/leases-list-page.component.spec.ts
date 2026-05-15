import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { LeasesListPageComponent } from './leases-list-page.component'
import { UntypedFormBuilder } from '@angular/forms'
import { ConfirmationService, MessageService } from 'primeng/api'
import { DHCPService, ServicesService } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideRouter } from '@angular/router'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('LeasesListsPageComponent', () => {
    let component: LeasesListPageComponent
    let fixture: ComponentFixture<LeasesListPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                UntypedFormBuilder,
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([]),
                {
                    provide: ServicesService,
                    useValue: { getDaemonsDirectory: () => of({ items: [{ id: 1, label: 'daemon' }], total: 1 }) },
                },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LeasesListPageComponent)
        component = fixture.componentInstance

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Leases List')
    })
})
