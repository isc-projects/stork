import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormGroup, Validators } from '@angular/forms'
import { Router } from '@angular/router'

import { MessageService } from 'primeng/api'
import { PasswordModule } from 'primeng/password'

import { UsersService } from '../backend/api/api'
import { AuthService } from '../auth.service'

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

    passwordChangeForm: FormGroup

    constructor(
        private router: Router,
        private formBuilder: FormBuilder,
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
                detail: 'New password must be different than the current password.',
                sticky: false,
            })
            return
        }

        // Send the old and new password to the server.
        this.usersApi.updateUserPassword(id, passwords).subscribe(
            (data) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User password updated',
                })
            },
            (err) => {
                console.info(err)
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
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
