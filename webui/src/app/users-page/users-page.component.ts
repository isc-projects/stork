import { Component, computed, model, Input, OnDestroy, OnInit, output, effect, viewChild, input } from '@angular/core'
import { FormControl, ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { ConfirmationService, MessageService, SelectItem } from 'primeng/api'

import { AuthService } from '../auth.service'
import { ServerDataService } from '../server-data.service'
import { UsersService } from '../backend'
import { debounceTime, firstValueFrom, lastValueFrom, Subject, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Group, User } from '../backend'
import { FormState, TabType, TabViewComponent } from '../tab-view/tab-view.component'
import { Panel } from 'primeng/panel'
import { Message } from 'primeng/message'
import { Select } from 'primeng/select'
import { Password } from 'primeng/password'
import { Checkbox } from 'primeng/checkbox'
import { ManagedAccessDirective } from '../managed-access.directive'
import { Button } from 'primeng/button'
import { InputText } from 'primeng/inputtext'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Table } from 'primeng/table'
import { Router } from '@angular/router'
import { distinctUntilChanged, map } from 'rxjs/operators'

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
 * Indicates if the user in an active tab is managed by an internal
 * authentication service
 */
export function isInternalUser(user: User) {
    const authenticationMethodId = user.authenticationMethodId
    // Empty or null or internal.
    return !authenticationMethodId || authenticationMethodId === 'internal'
}

/**
 * Component for managing system users.
 */
@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass'],
})
export class UsersPageComponent implements OnInit, OnDestroy {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Users' }]

    groups: Group[] = []
    // users table
    users: User[] = []
    totalUsers: number = 0

    tabView = viewChild(TabViewComponent)

    table = viewChild(Table)

    userProvider: (id: number) => Promise<User> = (id) => lastValueFrom(this.usersApi.getUser(id))

    userFormProvider = () => new UserFormState()

    tabTitleProvider: (user: User) => string = (user: User) => user.login || user.email

    private _subscriptions: Subscription = new Subscription()

    constructor(
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        public auth: AuthService,
        private confirmService: ConfirmationService,
        private router: Router
    ) {}

    ngOnInit() {
        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    map((f) => {
                        return { ...f, value: f.value ?? null }
                    }),
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        // this.zone.run(() =>
                        this.router.navigate(
                            [],
                            { queryParams: tableFiltersToQueryParams(this.table()) }
                            // )
                        )
                    })
                )
                .subscribe()
        )

        firstValueFrom(this.serverData.getGroups()).then((groups) => (this.groups = groups.items ?? []))
    }

    ngOnDestroy() {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Loads system users from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned and the filter text.
     */
    loadUsers(event) {
        lastValueFrom(this.usersApi.getUsers(event.first, event.rows, event.filters['text'].value || null))
            .then((data) => {
                this.users = data.items ?? []
                this.totalUsers = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Loading user accounts failed',
                    detail: 'Failed to load user accounts from the database: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Displays a dialog to confirm user deletion.
     * @param id
     */
    confirmDeleteUser(id: number) {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this user?',
            header: 'Delete User',
            icon: 'pi pi-exclamation-triangle',
            rejectButtonProps: { text: true },
            accept: () => {
                this.deleteUser(id)
            },
        })
    }

    /**
     * Action invoked when existing user form is being deleted
     *
     * As a result of this action an existing user account is attempted to be
     * deleted.
     */
    deleteUser(id: number) {
        lastValueFrom(this.usersApi.deleteUser(id))
            .then((/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User account deleted',
                    detail: 'Successfully deleted user account.',
                })
                this.tabView()?.onDeleteEntity(id)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to delete user account',
                    detail: 'Failed to delete user account: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Return group name for the particular group id
     *
     * @param groupId group id for which the name should be returned.
     * @returns group name.
     */
    public getGroupName(groupId): string {
        // The super-admin group is well known and doesn't require
        // iterating over the list of groups fetched from the server.
        // Especially, if the server didn't respond properly for
        // some reason, we still want to be able to handle the
        // super-admin group.
        if (groupId === 1) {
            return 'super-admin'
        }
        for (const grp of this.groups) {
            if (grp.id === groupId) {
                return grp.name
            }
        }
        return 'unknown'
    }

    protected readonly isInternalUser = isInternalUser
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears the PrimeNG table state (filtering, pagination are reset).
     */
    clearTableState() {
        this.table()?.clear()
        this.router.navigate([])
    }

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     *
     * @param value
     * @param filterConstraint
     * @param debounceMode
     */
    filterTable(value: any, filterConstraint: FilterMetadata, debounceMode = true): void {
        if (debounceMode) {
            this._tableFilter$.next({ value, filterConstraint })
            return
        }

        filterConstraint.value = value
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table()) })
    }

    /**
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table()) })
    }
}

class UserFormState implements FormState {
    /**
     * Not used in this form
     */
    transactionID: number = 0

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the HostFormComponent form).
     */
    group: UntypedFormGroup

    user: User
}

@Component({
    selector: 'app-user-form',
    templateUrl: './user-form.component.html',
    standalone: true,
    imports: [
        Panel,
        Message,
        ReactiveFormsModule,
        Select,
        Password,
        Checkbox,
        ManagedAccessDirective,
        Button,
        InputText,
    ],
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
                    userPassword: ['', [Validators.minLength(8), Validators.maxLength(this.maxInputLen)]],
                    userPassword2: ['', [Validators.minLength(8), Validators.maxLength(this.maxInputLen)]],
                    changePassword: [''],
                },
                {
                    validators: [matchPasswords('userPassword', 'userPassword2')],
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
                    userPassword: [
                        '',
                        [Validators.required, Validators.minLength(8), Validators.maxLength(this.maxInputLen)],
                    ],
                    userPassword2: [
                        '',
                        [Validators.required, Validators.minLength(8), Validators.maxLength(this.maxInputLen)],
                    ],
                    changePassword: [true],
                },
                {
                    validators: [matchPasswords('userPassword', 'userPassword2')],
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
     * @param comparePasswords when true, feedback about passwords mismatch is also appended; defaults to false
     */
    buildFeedbackMessage(name: string, formatFeedback?: string, comparePasswords = false): string | null {
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

        if (comparePasswords && this.formGroup.errors?.['mismatchedPasswords']) {
            errors.push('Passwords must match.')
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
        return (
            (this.formGroup.get(name).invalid ||
                (comparePasswords && this.formGroup.errors?.['mismatchedPasswords'])) &&
            (this.formGroup.get(name).dirty || this.formGroup.get(name).touched)
        )
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
