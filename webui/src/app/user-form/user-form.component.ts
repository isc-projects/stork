import { Component, computed, Input, model, OnInit, input, effect, output } from '@angular/core'
import { Button } from 'primeng/button'
import { Checkbox } from 'primeng/checkbox'
import { InputText } from 'primeng/inputtext'
import { ManagedAccessDirective } from '../managed-access.directive'
import { Message } from 'primeng/message'
import { Panel } from 'primeng/panel'
import { Password } from 'primeng/password'
import {
    FormControl,
    ReactiveFormsModule,
    UntypedFormBuilder,
    UntypedFormGroup,
    ValidatorFn,
    Validators,
} from '@angular/forms'
import { Select } from 'primeng/select'
import { Group, User, UsersService } from '../backend'
import { UserFormState } from '../forms/user-form'
import { isInternalUser } from '../users-page/users-page.component'
import { MessageService, SelectItem } from 'primeng/api'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { TabType } from '../tab-view/tab-view.component'
import { StorkValidators } from '../validators'

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
export function matchPasswords(passwordKey: string, confirmPasswordKey: string) {
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
export function differentPasswords(oldPasswordKey: string, newPasswordKey: string) {
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
export function validatorPassword(): ValidatorFn | null {
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
export function validatorsConfirmPassword(
    oldPasswordKey: string | null,
    newPasswordKey: string,
    confirmPasswordKey: string
): ValidatorFn[] {
    const validators: ValidatorFn[] = [matchPasswords(newPasswordKey, confirmPasswordKey)]
    if (oldPasswordKey) {
        validators.push(differentPasswords(oldPasswordKey, newPasswordKey))
    }
    return validators
}

/**
 * Extracts errors from the given input and formats them into
 * human-readable messages.
 */
export function formatPasswordErrors(name: string, group: UntypedFormGroup, comparePasswords = false): string[] {
    const control = group.get(name)
    const errors: string[] = []

    if (control.errors?.['required']) {
        errors.push('This field is required.')
    }

    if (control.errors?.['minlength']) {
        errors.push('This field value must be at least 12 characters long.')
    }

    if (control.errors?.['maxlength']) {
        errors.push('This field value must be at most 120 characters long.')
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
export function isPasswordFeedbackNeeded(name: string, group: UntypedFormGroup, comparePasswords = false): boolean {
    return (
        (group.get(name).invalid ||
            (comparePasswords && group.errors?.['mismatchedPasswords']) ||
            (comparePasswords && group.errors?.['samePasswords'])) &&
        (group.get(name).dirty || group.get(name).touched)
    )
}

@Component({
    selector: 'app-user-form',
    imports: [
        Button,
        Checkbox,
        InputText,
        ManagedAccessDirective,
        Message,
        Panel,
        Password,
        ReactiveFormsModule,
        Select,
    ],
    templateUrl: './user-form.component.html',
    styleUrl: './user-form.component.sass',
})
export class UserFormComponent implements OnInit {
    @Input() formState: UserFormState = null

    user = model<User>()

    isInternalUser = computed(() => (this.user() ? isInternalUser(this.user()) : true))

    @Input() tabType!: TabType

    groups = input<Group[]>([])

    /**
     * RegExp pattern to validate password fields.
     * It allows uppercase and lowercase letters A-Z,
     * numbers 0-9, all special characters and whitespace
     * characters (i.e., space, tab, form feed, and line feed).
     */
    passwordPattern: RegExp = /^[a-zA-Z0-9~`!@#$%^&*()_+\-=\[\]\\{}|;':",.\/<>?\s]+$/

    // form data
    userGroups = computed<SelectItem[]>(() => {
        const initUserGroups = [
            {
                label: 'Select Group',
                value: null,
            },
        ]
        this.groups().forEach((group: Group) => {
            initUserGroups.push({
                label: group.name,
                value: { id: group.id, name: group.name },
            })
        })
        return [...initUserGroups]
    })

    constructor(
        private _formBuilder: UntypedFormBuilder,
        private usersApi: UsersService,
        private messageService: MessageService
    ) {
        effect(() => {
            this.formState.user = this.user()
        })
    }

    /**
     * Returns main form group for the component.
     *
     * @returns form group.
     */
    get formGroup(): UntypedFormGroup {
        return this.formState.group
    }

    /**
     * Sets main form group for the component.
     *
     * @param fg new form group.
     */
    set formGroup(fg: UntypedFormGroup) {
        this.formState.group = fg
    }

    /**
     * Max input length allowed to be provided by a user. This is used in form validation.
     */
    maxInputLen = 120

    ngOnInit() {
        if (this.user()) {
            // Edit user form.
            this.formState.transactionID = this.user().id

            this.formGroup = this._formBuilder.group(
                {
                    userLogin: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                    userEmail: ['', [Validators.email, Validators.maxLength(this.maxInputLen)]],
                    userFirst: ['', Validators.maxLength(this.maxInputLen)],
                    userLast: ['', Validators.maxLength(this.maxInputLen)],
                    userGroup: ['', Validators.required],
                    userPassword: ['', [validatorPassword()]],
                    userPassword2: ['', [validatorPassword()]],
                    changePassword: [''],
                },
                {
                    validators: validatorsConfirmPassword(null, 'userPassword', 'userPassword2'),
                }
            )

            // The authentication hooks may not support returning profile details
            // as email, first and last names, or groups.
            if (this.isInternalUser()) {
                this.formGroup.setControl(
                    'userFirst',
                    new FormControl('', [Validators.required, Validators.maxLength(this.maxInputLen)])
                )
                this.formGroup.setControl(
                    'userLast',
                    new FormControl('', [Validators.required, Validators.maxLength(this.maxInputLen)])
                )
            }

            this.formGroup.patchValue({
                userLogin: this.user().login,
                userEmail: this.user().email,
                userFirst: this.user().name,
                userLast: this.user().lastname,
                changePassword: this.user().changePassword,
            })

            if (this.groups().length && this.user().groups && this.user().groups.length) {
                this.formGroup.patchValue({
                    userGroup: {
                        id: this.groups()[this.user().groups[0] - 1].id,
                        name: this.groups()[this.user().groups[0] - 1].name,
                    },
                })
            }
        } else {
            // New user form.
            this.formState.transactionID = -1

            this.formGroup = this._formBuilder.group(
                {
                    userLogin: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                    userEmail: ['', [Validators.email, Validators.maxLength(this.maxInputLen)]],
                    userFirst: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                    userLast: ['', [Validators.required, Validators.maxLength(this.maxInputLen)]],
                    userGroup: ['', Validators.required],
                    userPassword: ['', [Validators.required, validatorPassword()]],
                    userPassword2: ['', [Validators.required, validatorPassword()]],
                    changePassword: [true],
                },
                {
                    validators: validatorsConfirmPassword(null, 'userPassword', 'userPassword2'),
                }
            )
        }
    }

    protected readonly TabType = TabType

    /**
     * Utility function which builds feedback message when form field validation failed.
     *
     * @param name FormControl name for which the feedback is to be generated
     * @param formatFeedback optional feedback message when pattern validation failed
     */
    buildFeedbackMessage(name: string, formatFeedback?: string, comparePasswords = false): string | null {
        if (name === 'userPassword' || name === 'userPassword2') {
            return formatPasswordErrors(name, this.formGroup, comparePasswords).join(' ')
        }

        const errors: string[] = []

        if (this.formGroup.get(name).errors?.['required']) {
            errors.push('This field is required.')
        }

        if (this.formGroup.get(name).errors?.['minlength']) {
            errors.push('This field value is too short.')
        }

        if (this.formGroup.get(name).errors?.['maxlength']) {
            errors.push('This field value is too long.')
        }

        if (this.formGroup.get(name).errors?.['email']) {
            errors.push('Email is invalid.')
        }

        if (this.formGroup.get(name).errors?.['pattern']) {
            errors.push(formatFeedback ?? 'This field value is incorrect.')
        }

        return errors.join(' ')
    }

    /**
     * Utility function which checks if feedback for given FormControl shall be displayed.
     *
     * @param name FormControl name for which the check is done
     * @param comparePasswords when true, passwords mismatch is also checked; defaults to false
     */
    isFeedbackNeeded(name: string, comparePasswords = false): boolean {
        if (name === 'userPassword' || name === 'userPassword2') {
            return isPasswordFeedbackNeeded(name, this.formGroup, comparePasswords)
        }

        return this.formGroup.get(name).invalid && (this.formGroup.get(name).dirty || this.formGroup.get(name).touched)
    }

    /**
     * Returns group description for given group ID.
     * @param groupId numeric group ID
     */
    public getGroupDescription(groupId: number): string {
        return this.groups().find((group) => group.id === groupId)?.description
    }

    /**
     * Action invoked when new user form is being saved
     *
     * As a result of this action a new user account is attempted to be
     * created.
     */
    private newUserSave() {
        const user = {
            id: 0,
            login: this.formGroup.controls.userLogin.value,
            email: this.formGroup.controls.userEmail.value,
            name: this.formGroup.controls.userFirst.value,
            lastname: this.formGroup.controls.userLast.value,
            groups: [this.formGroup.controls.userGroup.value.id],
            authenticationMethodId: '',
            changePassword: this.formGroup.controls.changePassword.value,
        }
        const password = this.formGroup.controls.userPassword.value
        const account = { user, password }
        lastValueFrom(this.usersApi.createUser(account))
            .then((user) => {
                this.messageService.add({
                    severity: 'success',
                    summary: 'New user account created',
                    detail: 'Successfully added new user account.',
                })
                this.user.set(user)
                this.formState.user = user
                this.formSubmit.emit(this.formState)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to create new user account',
                    detail: 'Failed to create new user account: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Action invoked when the form for editing the user information is saved
     *
     * As a result of this action, the user account information will be updated.
     */
    private editedUserSave() {
        const user = {
            id: this.user().id,
            login: this.formGroup.controls.userLogin.value,
            email: this.formGroup.controls.userEmail.value,
            name: this.formGroup.controls.userFirst.value,
            lastname: this.formGroup.controls.userLast.value,
            groups: [this.formGroup.controls.userGroup.value.id],
            authenticationMethodId: '',
            changePassword: this.formGroup.controls.changePassword.value,
        }
        const password = this.formGroup.controls.userPassword.value
        const account = { user, password }
        lastValueFrom(this.usersApi.updateUser(account))
            .then(() => {
                this.messageService.add({
                    severity: 'success',
                    summary: 'User account updated',
                    detail: 'Successfully updated user account.',
                })
                this.formSubmit.emit(this.formState)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to update user account',
                    detail: 'Failed to update user account: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Action invoked when a user form is saved
     *
     * It covers both the case of creating a new user account and editing
     * an existing user account.
     */
    userFormSave() {
        if (this.tabType === TabType.New) {
            this.newUserSave()
        } else if (this.tabType === TabType.Edit) {
            this.editedUserSave()
        }
    }

    userFormCancel() {
        this.formCancel.emit()
    }

    formSubmit = output<UserFormState>()

    formCancel = output()
}
