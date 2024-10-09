import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { UsersService } from '../backend/api/api'
import { AuthService } from '../auth.service'
import { getErrorMessage } from '../utils'
import { matchPasswords } from '../users-page/users-page.component'

/**
 * This component allows the logged user to change the password.
 */
@Component({
    selector: 'app-settings-page',
    templateUrl: './password-change-page.component.html',
    styleUrls: ['./password-change-page.component.sass'],
})
export class PasswordChangePageComponent implements OnInit {
    breadcrumbs = [{ label: 'User Profile' }, { label: 'Password Change' }]

    passwordChangeForm: UntypedFormGroup

    /**
     * Max input length allowed to be provided by a user. This is used in form validation.
     */
    maxInputLen = 120

    /**
     * RegExp pattern to validate password fields.
     * It allows uppercase and lowercase letters A-Z,
     * numbers 0-9 and all special characters.
     */
    passwordPattern: RegExp = /^[a-zA-Z0-9~`!@#$%^&*()_+\-=\[\]\\{}|;':",.\/<>?\s]+$/

    constructor(
        private formBuilder: UntypedFormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private auth: AuthService
    ) {}

    ngOnInit() {
        this.passwordChangeForm = this.formBuilder.group(
            {
                oldPassword: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                newPassword: [
                    '',
                    [Validators.required, Validators.minLength(8), Validators.maxLength(this.maxInputLen)],
                ],
                confirmPassword: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
            },
            { validators: [matchPasswords('newPassword', 'confirmPassword')] }
        )
    }

    /**
     * Indicates if the user was authenticated by the external authentication
     * service.
     */
    get isExternalUser(): boolean {
        return !this.auth.isInternalUser()
    }

    /**
     * Action invoked upon password change submission.
     *
     * Sends the old and new password to the server for update. The old
     * password is used for authorization. If the old and new password
     * are the same, an error is displayed.
     */
    passwordChangeFormSubmit() {
        const id = this.auth.currentUserValue.id
        const passwords = {
            oldpassword: this.passwordChangeForm.controls.oldPassword.value,
            newpassword: this.passwordChangeForm.controls.newPassword.value,
        }

        // Do not contact the server if the new password is the same.
        if (passwords.oldpassword === passwords.newpassword) {
            this.msgSrv.add({
                severity: 'warn',
                summary: 'Password not updated',
                detail: 'New password must be different from the current password.',
                sticky: false,
            })
            return
        }

        // Send the old and new password to the server.
        this.usersApi.updateUserPassword(id, passwords).subscribe(
            (/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User password updated',
                })
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Updating user password failed',
                    detail: msg,
                    sticky: true,
                })
            }
        )
    }

    /**
     * Utility function which checks if feedback for given FormControl shall be displayed.
     *
     * @param name FormControl name for which the check is done
     * @param comparePasswords when true, passwords mismatch is also checked; defaults to false
     */
    isFeedbackNeeded(name: string, comparePasswords = false): boolean {
        return (
            (this.passwordChangeForm.get(name).invalid ||
                (comparePasswords && this.passwordChangeForm.errors?.['mismatchedPasswords'])) &&
            (this.passwordChangeForm.get(name).dirty || this.passwordChangeForm.get(name).touched)
        )
    }

    /**
     * Utility function which builds feedback message when form field validation failed.
     *
     * @param name FormControl name for which the feedback is to be generated
     * @param formatFeedback optional feedback message when pattern validation failed
     * @param comparePasswords when true, feedback about passwords mismatch is also appended; defaults to false
     */
    buildFeedbackMessage(name: string, formatFeedback?: string, comparePasswords = false): string | null {
        const errors: string[] = []

        if (this.passwordChangeForm.get(name).errors?.['required']) {
            errors.push('This field is required.')
        }

        if (this.passwordChangeForm.get(name).errors?.['minlength']) {
            errors.push('This field value is too short.')
        }

        if (this.passwordChangeForm.get(name).errors?.['maxlength']) {
            errors.push('This field value is too long.')
        }

        if (this.passwordChangeForm.get(name).errors?.['pattern']) {
            errors.push(formatFeedback ?? 'This field value is wrong.')
        }

        if (comparePasswords && this.passwordChangeForm.errors?.['mismatchedPasswords']) {
            errors.push('Passwords must match.')
        }

        return errors.join(' ')
    }
}
