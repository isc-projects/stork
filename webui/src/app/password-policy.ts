import { UntypedFormGroup, ValidatorFn, Validators } from '@angular/forms'
import { StorkValidators } from './validators'

export class PasswordPolicy {
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
        const validators: ValidatorFn[] = [StorkValidators.areSame(newPasswordKey, confirmPasswordKey)]
        if (oldPasswordKey) {
            validators.push(StorkValidators.areNotSame(oldPasswordKey, newPasswordKey))
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
            if (group.errors?.['areNotSame']) {
                errors.push('Passwords must match.')
            }

            if (group.errors?.['areSame']) {
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
                (comparePasswords && group.errors?.['areNotSame']) ||
                (comparePasswords && group.errors?.['areSame'])) &&
            (group.get(name).dirty || group.get(name).touched)
        )
    }
}
