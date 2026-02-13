import { FormControl, FormGroup } from '@angular/forms'
import { PasswordPolicy } from './password-policy'

describe('PasswordPolicy', () => {
    it('should recognize invalid password', () => {
        const validator = PasswordPolicy.validatorPassword()
        const control: FormControl = new FormControl('', validator)
        const group: FormGroup = new FormGroup({ password: control })
        control.markAsDirty()

        // Empty new password.
        control.setValue('')
        control.updateValueAndValidity()
        expect(control.valid).toBeTrue()
        expect(control.errors).toBeNull()
        expect(control.errors?.['required']).toBeUndefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeFalse()
        expect(PasswordPolicy.formatPasswordErrors('password', group).length).toBe(0)

        // Minimum length violation.
        control.setValue('Short1!')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['minlength']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must be at least 12 characters long.'
        )

        // Maximum length violation.
        const longPassword = 'A'.repeat(121)
        control.setValue(longPassword)
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['maxlength']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must be at most 120 characters long.'
        )

        // Missing uppercase letter.
        control.setValue('lowercase123!')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['hasUppercaseLetter']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must contain at least one uppercase letter.'
        )

        // Missing lowercase letter.
        control.setValue('UPPERCASE123!')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['hasLowercaseLetter']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must contain at least one lowercase letter.'
        )

        // Missing digit.
        control.setValue('NoDigitsHere!')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['hasDigit']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must contain at least one digit.'
        )

        // Missing special character.
        control.setValue('NoSpecialChar1')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['hasSpecialCharacter']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('password', group)[0]).toBe(
            'Password must contain at least one special character or whitespace.'
        )
        // Many violations at once.
        control.setValue('short')
        control.updateValueAndValidity()
        expect(control.valid).toBeFalse()
        expect(control.errors).not.toBeNull()
        expect(control.errors['minlength']).toBeDefined()
        expect(control.errors['hasUppercaseLetter']).toBeDefined()
        expect(control.errors['hasDigit']).toBeDefined()
        expect(control.errors['hasSpecialCharacter']).toBeDefined()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeTrue()
        const errors = PasswordPolicy.formatPasswordErrors('password', group)
        expect(errors).toContain('Password must be at least 12 characters long.')
        expect(errors).toContain('Password must contain at least one uppercase letter.')
        expect(errors).toContain('Password must contain at least one digit.')
        expect(errors).toContain('Password must contain at least one special character or whitespace.')

        // Valid password.
        control.setValue('ValidPassword123!')
        control.updateValueAndValidity()
        expect(control.valid).toBeTrue()
        expect(control.errors).toBeNull()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('password', group)).toBeFalse()
        expect(PasswordPolicy.formatPasswordErrors('password', group).length).toBe(0)
    })

    it('should verify if the new password is different from the old one', () => {
        const group = new FormGroup(
            {
                oldPassword: new FormControl('password'),
                newPassword: new FormControl('password'),
                confirmPassword: new FormControl('password'),
            },
            { validators: PasswordPolicy.validatorsConfirmPassword('newPassword', 'confirmPassword', 'oldPassword') }
        )
        group.get('oldPassword').markAsDirty()
        group.get('newPassword').markAsDirty()
        group.updateValueAndValidity()

        expect(PasswordPolicy.isPasswordFeedbackNeeded('newPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('newPassword', group, true)[0]).toBe(
            'New password must be different from current password.'
        )

        group.get('newPassword')?.setValue('another-password')
        group.get('confirmPassword')?.setValue('another-password')
        group.updateValueAndValidity()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('oldPassword', group, true)).toBeFalse()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('newPassword', group, true)).toBeFalse()
    })

    it('should verify if the new password is the same as the confirm one', () => {
        const group = new FormGroup(
            {
                oldPassword: new FormControl('old-password'),
                newPassword: new FormControl('password'),
                confirmPassword: new FormControl('password'),
            },
            { validators: PasswordPolicy.validatorsConfirmPassword('newPassword', 'confirmPassword', 'oldPassword') }
        )
        group.get('newPassword').markAsDirty()
        group.get('confirmPassword').markAsDirty()
        group.updateValueAndValidity()

        expect(PasswordPolicy.isPasswordFeedbackNeeded('newPassword', group, true)).toBeFalse()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeFalse()

        group.get('confirmPassword')?.setValue('another-password')
        group.updateValueAndValidity()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('confirmPassword', group, true)[0]).toBe('Passwords must match.')
    })

    it('should verify the confirm password against the password policy when the old password is not checked', () => {
        const group = new FormGroup(
            {
                newPassword: new FormControl('ValidPassword123!'),
                confirmPassword: new FormControl('ValidPassword123!'),
            },
            PasswordPolicy.validatorsConfirmPassword('newPassword', 'confirmPassword')
        )
        group.get('newPassword').markAsDirty()
        group.get('confirmPassword').markAsDirty()
        group.updateValueAndValidity()

        expect(group.valid).toBeTrue()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('newPassword', group, true)).toBeFalse()
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeFalse()

        group.get('confirmPassword')?.setValue('invalid')
        group.updateValueAndValidity()
        expect(group.valid).toBeFalse()
        expect(group.errors).toEqual({ areSame: true })
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('confirmPassword', group, true)[0]).toBe('Passwords must match.')
    })

    it('should verify the confirm password against the password policy when the old password is checked', () => {
        const group = new FormGroup(
            {
                oldPassword: new FormControl('OldPassword123!'),
                newPassword: new FormControl('ValidPassword123!'),
                confirmPassword: new FormControl('ValidPassword123!'),
            },
            PasswordPolicy.validatorsConfirmPassword('newPassword', 'confirmPassword', 'oldPassword')
        )
        group.get('oldPassword').markAsDirty()
        group.get('newPassword').markAsDirty()
        group.get('confirmPassword').markAsDirty()
        group.updateValueAndValidity()

        expect(group.valid).toBeTrue()

        group.get('confirmPassword')?.setValue('invalid')
        group.updateValueAndValidity()
        expect(group.valid).toBeFalse()
        expect(group.errors).toEqual({ areSame: true })
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('confirmPassword', group, true)[0]).toBe('Passwords must match.')

        group.get('newPassword')?.setValue('OldPassword123!')
        group.updateValueAndValidity()
        expect(group.valid).toBeFalse()
        expect(group.errors).toEqual({ areSame: true, areNotSame: true })
        expect(PasswordPolicy.isPasswordFeedbackNeeded('newPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('newPassword', group, true)).toContain(
            'New password must be different from current password.'
        )
        expect(PasswordPolicy.isPasswordFeedbackNeeded('confirmPassword', group, true)).toBeTrue()
        expect(PasswordPolicy.formatPasswordErrors('confirmPassword', group, true)).toContain('Passwords must match.')
    })
})
