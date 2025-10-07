import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ActivatedRoute, RouterModule } from '@angular/router'
import { UntypedFormBuilder, ReactiveFormsModule } from '@angular/forms'
import { UsersService } from '../backend'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { PanelModule } from 'primeng/panel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { SettingsMenuComponent } from '../settings-menu/settings-menu.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { MenuModule } from 'primeng/menu'
import { PasswordModule } from 'primeng/password'
import { MessageModule } from 'primeng/message'
import { AuthService } from '../auth.service'
import { DialogModule } from 'primeng/dialog'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [PasswordChangePageComponent, BreadcrumbsComponent, SettingsMenuComponent, HelpTipComponent],
            imports: [
                PanelModule,
                NoopAnimationsModule,
                BreadcrumbModule,
                PopoverModule,
                MenuModule,
                RouterModule,
                ReactiveFormsModule,
                PasswordModule,
                MessageModule,
                DialogModule,
            ],
            providers: [
                UntypedFormBuilder,
                UsersService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
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
