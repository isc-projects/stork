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

    it('should recognize invalid password', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('admin')

        // Empty new password.
        component.passwordChangeForm.get('newPassword').setValue('')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['required']).not.toBeNull()

        // Minimum length violation.
        component.passwordChangeForm.get('newPassword').setValue('Short1!')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['minlength']).not.toBeNull()

        // Maximum length violation.
        const longPassword = 'A'.repeat(121)
        component.passwordChangeForm.get('newPassword').setValue(longPassword)
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['maxlength']).not.toBeNull()

        // Missing uppercase letter.
        component.passwordChangeForm.get('newPassword').setValue('lowercase123!')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasUppercaseLetter']).not.toBeNull()

        // Missing lowercase letter.
        component.passwordChangeForm.get('newPassword').setValue('UPPERCASE123!')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasLowercaseLetter']).not.toBeNull()

        // Missing digit.
        component.passwordChangeForm.get('newPassword').setValue('NoDigitsHere!')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasDigit']).not.toBeNull()

        // Missing special character.
        component.passwordChangeForm.get('newPassword').setValue('NoSpecialChar1')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasSpecialCharacter']).not.toBeNull()

        // Many violations at once.
        component.passwordChangeForm.get('newPassword').setValue('short')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeFalse()
        expect(component.passwordChangeForm.get('newPassword').errors).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['minlength']).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasUppercaseLetter']).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasDigit']).not.toBeNull()
        expect(component.passwordChangeForm.get('newPassword').errors['hasSpecialCharacter']).not.toBeNull()

        // Valid password.
        component.passwordChangeForm.get('newPassword').setValue('ValidPassword123!')
        component.passwordChangeForm.get('confirmPassword').setValue('ValidPassword123!')
        fixture.detectChanges()
        expect(component.passwordChangeForm.valid).toBeTrue()
    })

    it('should permit spaces in the password', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('admin')
        component.passwordChangeForm.get('newPassword').setValue('Password with spaces works well in 2026!')
        component.passwordChangeForm.get('confirmPassword').setValue('Password with spaces works well in 2026!')

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

    it('should return required error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('')
        component.passwordChangeForm.get('oldPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('oldPassword')
        expect(message).toContain('This field is required.')
    })

    it('should return minlength error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('Short1!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('This field value must be at least 12 characters long.')
    })

    it('should return maxlength error message', () => {
        component.ngOnInit()
        const longPassword = 'A'.repeat(121)
        component.passwordChangeForm.get('oldPassword').setValue(longPassword)
        component.passwordChangeForm.get('oldPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('oldPassword')
        expect(message).toContain('This field value must be at most 120 characters long.')
    })

    it('should return uppercase letter error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('lowercase123!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('Password must contain at least one uppercase letter.')
    })

    it('should return lowercase letter error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('UPPERCASE123!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('Password must contain at least one lowercase letter.')
    })

    it('should return digit error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('NoDigitsHere!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('Password must contain at least one digit.')
    })

    it('should return special character error message', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('NoSpecialChar1')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('Password must contain at least one special character.')
    })

    it('should return mismatched passwords error message when comparePasswords is true', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('ValidPassword123!')
        component.passwordChangeForm.get('confirmPassword').setValue('DifferentPassword123!')
        component.passwordChangeForm.get('confirmPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('confirmPassword', undefined, true)
        expect(message).toContain('Passwords must match.')
    })

    it('should return same passwords error message when comparePasswords is true', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('SamePassword123!')
        component.passwordChangeForm.get('newPassword').setValue('SamePassword123!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword', undefined, true)
        expect(message).toContain('New password must be different from current password.')
    })

    it('should return multiple error messages when multiple validations fail', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('newPassword').setValue('short')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword')
        expect(message).toContain('This field value must be at least 12 characters long.')
        expect(message).toContain('Password must contain at least one uppercase letter.')
        expect(message).toContain('Password must contain at least one digit.')
        expect(message).toContain('Password must contain at least one special character.')
    })

    it('should return empty string when no errors', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('ValidPassword123!')
        component.passwordChangeForm.get('oldPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('oldPassword')
        expect(message).toBe('')
    })

    it('should not include password comparison errors when comparePasswords is false', () => {
        component.ngOnInit()
        component.passwordChangeForm.get('oldPassword').setValue('SamePassword123!')
        component.passwordChangeForm.get('newPassword').setValue('SamePassword123!')
        component.passwordChangeForm.get('confirmPassword').setValue('DifferentPassword123!')
        component.passwordChangeForm.get('newPassword').markAsTouched()
        fixture.detectChanges()

        const message = component.buildFeedbackMessage('newPassword', undefined, false)
        expect(message).not.toContain('New password must be different from current password.')
        expect(message).not.toContain('Passwords must match.')
    })
})
