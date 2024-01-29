import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { UsersService } from '../backend/api/api'
import { AuthService } from '../auth.service'
import { getErrorMessage } from '../utils'

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

    constructor(
        private formBuilder: UntypedFormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private auth: AuthService
    ) {}

    ngOnInit() {
        this.passwordChangeForm = this.formBuilder.group({
            oldPassword: ['', Validators.required],
            newPassword: ['', Validators.compose([Validators.required, Validators.minLength(8)])],
            confirmPassword: ['', Validators.required],
        })
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
}
