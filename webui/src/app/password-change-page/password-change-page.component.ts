import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormGroup, Validators } from '@angular/forms'
import { UrlTree, PRIMARY_OUTLET, Router } from '@angular/router'

import { MessageService } from 'primeng/api'
import { PasswordModule } from 'primeng/password'

import { UsersService } from '../backend/api/api'
import { AuthService } from '../auth.service'

@Component({
    selector: 'app-settings-page',
    templateUrl: './password-change-page.component.html',
    styleUrls: ['./password-change-page.component.sass'],
})
export class PasswordChangePageComponent implements OnInit {
    selectedItem: string

    passwordChangeForm: FormGroup

    constructor(
        private router: Router,
        private formBuilder: FormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private auth: AuthService
    ) {}

    ngOnInit() {
        const urlTree = this.router.parseUrl(this.router.url)
        const segments = urlTree.root.children[PRIMARY_OUTLET].segments
        this.selectedItem = 'profile'
        if (segments.length > 1) {
            this.selectedItem = segments[1].path
        }

        switch (this.selectedItem) {
            case 'password':
                this.passwordChangeForm = this.formBuilder.group({
                    oldPassword: ['', Validators.required],
                    newPassword: ['', Validators.compose([Validators.required, Validators.minLength(8)])],
                    confirmPassword: ['', Validators.required],
                })
                break
            default:
                break
        }
    }

    passwordChangeFormSubmit() {
        const id = this.auth.currentUserValue.id
        const passwords = {
            oldpassword: this.passwordChangeForm.controls.oldPassword.value,
            newpassword: this.passwordChangeForm.controls.newPassword.value,
        }

        if (passwords.oldpassword === passwords.newpassword) {
            this.msgSrv.add({
                severity: 'warn',
                summary: 'Password not updated',
                detail: 'New password must be different than the current password.',
                sticky: false,
            })
            return
        }

        this.usersApi.updateUserPassword(id, passwords).subscribe(
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User password updated',
                })
            },
            err => {
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
