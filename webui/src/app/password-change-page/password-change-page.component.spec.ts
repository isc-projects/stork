import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ActivatedRoute, provideRouter } from '@angular/router'
import { UntypedFormBuilder } from '@angular/forms'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { AuthService } from '../auth.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                UntypedFormBuilder,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(PasswordChangePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()

        authService = TestBed.inject(AuthService)
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
        expect(breadcrumbsComponent.items[0].label).toEqual('User Profile')
        expect(breadcrumbsComponent.items[1].label).toEqual('Password Change')
    })

    it('should permit spaces in the password', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('admin')
        component.passwordChangeForm.get('newPassword').setValue('password with spaces works well')
        component.passwordChangeForm.get('confirmPassword').setValue('password with spaces works well')

        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeTrue()
    })

    it('should recognize the password must be changed', () => {
        spyOnProperty(authService, 'currentUserValue').and.returnValues(
            {
                authenticationMethodId: 'internal',
                id: 1,
                changePassword: true,
            },
            {
                authenticationMethodId: 'internal',
                id: 1,
                changePassword: false,
            }
        )

        expect(component.mustChangePassword).toBeTrue()
        expect(component.mustChangePassword).toBeFalse()
    })
})
