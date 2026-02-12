import { UntypedFormGroup, ValidatorFn, Validators } from '@angular/forms'
import { StorkValidators } from './validators'

export class PasswordPolicy {
    /**
     * Form validator verifying if the confirmed password matches the password
     * value.
     *
     * @param passwordKey Name of the key under which the password value can be
     *                    found in the form.
     * @param confirmPasswordKey Name of the key under which the confirmed
     *                           password can be found in the form.
     * @returns The validator function comparing the passwords.
     */
    public static matchPasswords(passwordKey: string, confirmPasswordKey: string) {
        return (group: UntypedFormGroup): { [key: string]: any } => {
            const password = group.get(passwordKey)
            const confirmPassword = group.get(confirmPasswordKey)

            if (password?.value !== confirmPassword?.value) {
                return {
                    mismatchedPasswords: true,
                }
            }

            return null
        }
    }

    /**
     * Form validator verifying if the confirmed password is different from the
     * previous password.
     *
     * @param oldPasswordKey Name of the key under which the old password value can
     *                       be found in the form.
     * @param newPasswordKey Name of the key under which the new password value can
     *                       be found in the form.
     * @returns The validator function comparing the passwords.
     */
    public static differentPasswords(oldPasswordKey: string, newPasswordKey: string) {
        return (group: UntypedFormGroup): { [key: string]: any } => {
            const oldPassword = group.get(oldPasswordKey)
            const newPassword = group.get(newPasswordKey)

            if (oldPassword?.value === newPassword?.value) {
                return {
                    samePasswords: true,
                }
            }

            return null
        }
    }

    /**
     * A validator checking a new password against the password policy. The password must be between 12 and 120 characters
     * long and must contain at least one uppercase letter, one lowercase letter, one digit and one special character.
     */
    public static validatorPassword(): ValidatorFn | null {
        return Validators.compose([
            Validators.minLength(12),
            Validators.maxLength(120),
            StorkValidators.hasUppercaseLetter,
            StorkValidators.hasLowercaseLetter,
            StorkValidators.hasDigit,
            StorkValidators.hasSpecialCharacter,
        ])
    }

    /**
     * A validator checking a confirm password field. It must match the new password field and must not exceed the
     * maximum allowed length.
     * If the oldPasswordKey parameter is provided, it also checks that the new password is different from the current
     * password.
     */
    public static validatorsConfirmPassword(
        oldPasswordKey: string | null,
        newPasswordKey: string,
        confirmPasswordKey: string
    ): ValidatorFn[] {
        const validators: ValidatorFn[] = [PasswordPolicy.matchPasswords(newPasswordKey, confirmPasswordKey)]
        if (oldPasswordKey) {
            validators.push(PasswordPolicy.differentPasswords(oldPasswordKey, newPasswordKey))
        }
        return validators
    }

    /**
     * Extracts errors from the given input and formats them into
     * human-readable messages.
     */
    public static formatPasswordErrors(name: string, group: UntypedFormGroup, comparePasswords = false): string[] {
        const control = group.get(name)
        const errors: string[] = []

        if (control.errors?.['required']) {
            errors.push('Password is required.')
        }

        if (control.errors?.['minlength']) {
            errors.push('Password must be at least 12 characters long.')
        }

        if (control.errors?.['maxlength']) {
            errors.push('Password must be at most 120 characters long.')
        }

        if (control.errors?.['pattern']) {
            errors.push('Password must only contain letters, digits, special, or whitespace characters.')
        }

        if (control.errors?.['hasUppercaseLetter']) {
            errors.push('Password must contain at least one uppercase letter.')
        }

        if (control.errors?.['hasLowercaseLetter']) {
            errors.push('Password must contain at least one lowercase letter.')
        }

        if (control.errors?.['hasDigit']) {
            errors.push('Password must contain at least one digit.')
        }

        if (control.errors?.['hasSpecialCharacter']) {
            errors.push('Password must contain at least one special character.')
        }

        if (comparePasswords) {
            if (group.errors?.['mismatchedPasswords']) {
                errors.push('Passwords must match.')
            }

            if (group.errors?.['samePasswords']) {
                errors.push('New password must be different from current password.')
            }
        }

        return errors
    }

    /**
     * Utility function which checks if feedback for given FormControl shall be displayed.
     *
     * @param name FormControl name for which the check is done
     * @param comparePasswords when true, passwords mismatch is also checked; defaults to false
     */
    public static isPasswordFeedbackNeeded(name: string, group: UntypedFormGroup, comparePasswords = false): boolean {
        return !!(
            (group.get(name).invalid ||
                (comparePasswords && group.errors?.['mismatchedPasswords']) ||
                (comparePasswords && group.errors?.['samePasswords'])) &&
            (group.get(name).dirty || group.get(name).touched)
        )
    }
}
