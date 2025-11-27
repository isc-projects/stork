import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ProfilePageComponent } from './profile-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { User } from '../backend'
import { provideRouter } from '@angular/router'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('ProfilePageComponent', () => {
    let component: ProfilePageComponent
    let fixture: ComponentFixture<ProfilePageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ProfilePageComponent)
        component = fixture.componentInstance
        component.currentUser = {
            id: 1,
            email: 'user@example.com',
            login: 'foobar',
            name: 'foo',
            lastname: 'bar',
            groups: [],
            authenticationMethodId: 'internal',
        } as User
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
        expect(breadcrumbsComponent.items).toHaveSize(1)
        expect(breadcrumbsComponent.items[0].label).toEqual('User Profile')
    })

    it('should display not-specified placeholder if the external ID is empty', () => {
        const labels = fixture.debugElement.queryAll(By.css('div b'))
        const externalIdLabels = labels.filter((l) =>
            (l.nativeElement as HTMLElement).textContent.includes('External ID')
        )
        expect(externalIdLabels.length).toBe(1)
        const externalIdLabel = externalIdLabels[0]
        const externalIdValue = (externalIdLabel.nativeElement as HTMLElement).parentElement.nextElementSibling
        expect(externalIdValue.textContent).toContain('(not specified)')
    })

    it('should display the value if the external ID is not empty', () => {
        component.currentUser.externalId = 'foobar'
        fixture.detectChanges()

        const labels = fixture.debugElement.queryAll(By.css('div b'))
        const externalIdLabels = labels.filter((l) =>
            (l.nativeElement as HTMLElement).textContent.includes('External ID')
        )
        expect(externalIdLabels.length).toBe(1)
        const externalIdLabel = externalIdLabels[0]
        const externalIdValue = (externalIdLabel.nativeElement as HTMLElement).parentElement.nextElementSibling
        expect(externalIdValue.textContent).toContain('foobar')
    })
})
