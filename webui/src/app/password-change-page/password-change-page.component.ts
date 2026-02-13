import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators, FormsModule, ReactiveFormsModule } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { UsersService } from '../backend/api/api'
import { AuthService } from '../auth.service'
import { getErrorMessage } from '../utils'
import { ActivatedRoute, Router } from '@angular/router'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { SettingsMenuComponent } from '../settings-menu/settings-menu.component'
import { NgIf, NgTemplateOutlet } from '@angular/common'
import { Dialog } from 'primeng/dialog'
import { Message } from 'primeng/message'
import { Panel } from 'primeng/panel'
import { Password } from 'primeng/password'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { PasswordPolicy } from '../password-policy'

/**
 * This component allows the logged user to change the password.
 * The password policy defined in this file must match the one
 * implemented on the server side.
 * See backend/server/restservice/users.go: validatePassword function.
 */
@Component({
    selector: 'app-settings-page',
    templateUrl: './password-change-page.component.html',
    styleUrls: ['./password-change-page.component.sass'],
    imports: [
        BreadcrumbsComponent,
        SettingsMenuComponent,
        NgIf,
        NgTemplateOutlet,
        Dialog,
        Message,
        FormsModule,
        ReactiveFormsModule,
        Panel,
        Password,
        Button,
        ManagedAccessDirective,
    ],
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
     * numbers 0-9, all special characters and whitespace
     * characters (i.e., space, tab, form feed, and line feed).
     */
    passwordPattern: RegExp = /^[a-zA-Z0-9~`!@#$%^&*()_+\-=\[\]\\{}|;':",.\/<>?\s]+$/

    constructor(
        private formBuilder: UntypedFormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private auth: AuthService,
        private route: ActivatedRoute,
        private router: Router
    ) {}

    ngOnInit() {
        this.passwordChangeForm = this.formBuilder.group(
            {
                oldPassword: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                newPassword: ['', [Validators.required, PasswordPolicy.validatorPassword()]],
                confirmPassword: ['', [Validators.required, PasswordPolicy.validatorPassword()]],
            },
            {
                validators: PasswordPolicy.validatorsConfirmPassword('newPassword', 'confirmPassword', 'oldPassword'),
            }
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
     * Indicates if the user is forced to change the password.
     */
    get mustChangePassword(): boolean {
        return !!this.auth.currentUserValue?.changePassword
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
                detail: 'New password must be different from current password.',
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

                // Reset the change password flag.
                this.auth.resetChangePasswordFlag()

                // Redirect to the previous page if it was set.
                const returnUrl = this.route.snapshot.queryParams.returnUrl
                if (returnUrl) {
                    this.router.navigateByUrl(returnUrl)
                }
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to update user password',
                    detail: msg,
                    life: 10000,
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
        return PasswordPolicy.isPasswordFeedbackNeeded(name, this.passwordChangeForm, comparePasswords)
    }

    /**
     * Utility function which builds feedback message when form field validation failed.
     *
     * @param name FormControl name for which the feedback is to be generated
     */
    buildFeedbackMessage(name: string, comparePasswords = false): string | null {
        const errors = PasswordPolicy.formatPasswordErrors(name, this.passwordChangeForm, comparePasswords)
        return errors.join(' ')
    }
}
