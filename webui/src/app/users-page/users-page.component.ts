import { Component, OnDestroy, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators, FormControl } from '@angular/forms'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'
import { ConfirmationService, MenuItem, MessageService, SelectItem } from 'primeng/api'

import { AuthService } from '../auth.service'
import { ServerDataService } from '../server-data.service'
import { UsersService } from '../backend/api/api'
import { lastValueFrom, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { User } from '../backend'

/**
 * An enum specifying tab types in the user view
 *
 * Currently supported types are:
 * - list: including a list of users
 * - new user: including a form for creating a new user account
 * - edited user: including a form for editing a user account
 * - user: including read-only information about the user
 */
export enum UserTabType {
    List = 1,
    NewUser,
    EditedUser,
    User,
}

/**
 * Class representing a single tab on the user page
 */
export class UserTab {
    /**
     * Instance of the reactive form belonging to the tab
     */
    public userForm: UntypedFormGroup

    /**
     * Constructor
     */
    constructor(
        public tabType: UserTabType,
        public user: User
    ) {}

    /**
     * Returns route associated with this tab
     *
     * The returned value depends on the tab type.
     */
    get tabRoute(): string {
        switch (this.tabType) {
            case UserTabType.List: {
                return '/users/list'
            }
            case UserTabType.NewUser: {
                return '/users/new'
            }
            default: {
                if (this.user) {
                    return '/users/' + this.user.id
                }
            }
        }
        return '/users/list'
    }
}

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
 * Component for managing system users.
 */
@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass'],
})
export class UsersPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Users' }]

    /**
     * RegExp pattern to validate password fields.
     * It allows uppercase and lowercase letters A-Z,
     * numbers 0-9, all special characters and whitespace
     * characters (i.e., space, tab, form feed, and line feed).
     */
    passwordPattern: RegExp = /^[a-zA-Z0-9~`!@#$%^&*()_+\-=\[\]\\{}|;':",.\/<>?\s]+$/

    // ToDo: Strict typing
    private groups: any[] = []
    // users table
    users: any[]
    totalUsers: number
    userMenuItems: MenuItem[]

    // user tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    openedTabs: UserTab[]
    userTab: UserTab

    // form data
    userGroups: SelectItem[]

    /**
     * Max input length allowed to be provided by a user. This is used in form validation.
     */
    maxInputLen = 120

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: UntypedFormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        public auth: AuthService,
        private confirmService: ConfirmationService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Returns user form from the current tab.
     *
     * @returns instance of the form or null if the current tab includes no form.
     */
    get userForm(): UntypedFormGroup {
        return this.userTab ? this.userTab.userForm : null
    }

    /**
     * Checks if the current tab displays a single user
     *
     * @returns true if the current tab displays information about selected user
     */
    get existingUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.User
    }

    /**
     * Checks if the current tab contains a form to edit user information
     *
     * @returns true if the current tab contains a form
     */
    get editedUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.EditedUser
    }

    /**
     * Checks if the current tab is for creating new user account
     *
     * @returns true if the current tab is for creating new account
     */
    get newUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.NewUser
    }

    /**
     * Actives a tab with the given index
     */
    private switchToTab(index) {
        if (this.activeTabIdx !== index) {
            this.activeTabIdx = Number(index)
            if (index > 0) {
                this.userTab = this.openedTabs[index]
                this.router.navigate([this.userTab.tabRoute])
            } else {
                this.userTab = null
                this.router.navigate(['/users/list'])
            }
        }
        this.activeItem = this.tabs[index]
    }

    /**
     * Opens new tab of the specified type and switches to it
     *
     * @param tabType Enumeration indicating type of the new tab
     * @param user Structure holding user information
     */
    private addUserTab(tabType: UserTabType, user) {
        const userTab = new UserTab(tabType, user)
        this.openedTabs.push(userTab)
        // The new tab is now current one
        this.userTab = userTab
        this.tabs = [
            ...this.tabs,
            {
                label: tabType === UserTabType.NewUser ? 'New account' : user.login || user.email,
                routerLink: userTab.tabRoute,
            },
        ]
        this.switchToTab(this.tabs.length - 1)
    }

    /**
     * Turns the current user tab into a tab with the user editing form
     *
     * This function is invoked when the user clicks on Edit button in
     * the user tab.
     *
     * @param tab Current tab
     */
    editUserInfo(tab) {
        // Specify validators for the form. The last validator is our custom
        // validator which checks if the password and confirmed password
        // match. The validator allows leaving an empty password in which
        // case the password won't be modified.
        const userForm = this.formBuilder.group(
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
        if (this.isInternalUser) {
            userForm.setControl(
                'userFirst',
                new FormControl('', [Validators.required, Validators.maxLength(this.maxInputLen)])
            )
            userForm.setControl(
                'userLast',
                new FormControl('', [Validators.required, Validators.maxLength(this.maxInputLen)])
            )
        }

        // Modify the current tab type to 'edit'.
        tab.tabType = UserTabType.EditedUser

        // Set default values for the fields which may be edited by the user.
        userForm.patchValue({
            userLogin: this.userTab.user.login,
            userEmail: this.userTab.user.email,
            userFirst: this.userTab.user.name,
            userLast: this.userTab.user.lastname,
            changePassword: this.userTab.user.changePassword,
        })

        if (this.groups.length > 0 && this.userTab.user.groups && this.userTab.user.groups.length > 0) {
            userForm.patchValue({
                userGroup: {
                    id: this.groups[this.userTab.user.groups[0] - 1].id,
                    name: this.groups[this.userTab.user.groups[0] - 1].name,
                },
            })
        }

        this.userTab.userForm = userForm
    }

    /**
     * Opens a tab for creating new user account
     *
     * It first checks if such tab has been already opened and activates it
     * if it has. If the tab hasn't been opened this function will open it.
     */
    showNewUserTab() {
        // Specify the validators for the new user form. The last two
        // validators require password and confirmed password to exist.
        const userForm = this.formBuilder.group(
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

        // Search opened tabs for the 'New account' type.
        for (const i in this.openedTabs) {
            if (this.openedTabs[i].tabType === UserTabType.NewUser) {
                // The tab exists, simply activate it.
                this.switchToTab(i)
                return
            }
        }
        // The tab doesn't exist, so open it and activate it.
        this.addUserTab(UserTabType.NewUser, null)
        this.userTab.userForm = userForm
    }

    /**
     * Closes current tab
     */
    private closeActiveTab() {
        this.openedTabs.splice(this.activeTabIdx, 1)
        this.tabs = [...this.tabs.slice(0, this.activeTabIdx), ...this.tabs.slice(this.activeTabIdx + 1)]
        this.switchToTab(0)
    }

    /**
     * Closes selected tab
     */
    closeTab(event, idx) {
        const i = Number(idx)

        this.openedTabs.splice(i, 1)
        this.tabs = [...this.tabs.slice(0, i), ...this.tabs.slice(i + 1)]
        if (this.activeTabIdx === i) {
            this.switchToTab(i - 1)
        } else if (this.activeTabIdx > i) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    /**
     * Loads system users from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned and the filter text.
     */
    loadUsers(event) {
        lastValueFrom(this.usersApi.getUsers(event.first, event.rows, event.filters.text?.[0]?.value))
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
                    sticky: true,
                })
            })
    }

    /**
     * Initializes tabs depending on the active route
     *
     * The first/default tab is always opened. It comprises a list of users
     * which have an account in the system. If the active route specifies
     * any particular user id, a tab displaying the user information is also
     * opened.
     */
    ngOnInit() {
        this.users = []
        const initUserGroups = [
            {
                label: 'Select Group',
                value: null,
            },
        ]
        this.userGroups = [...initUserGroups]
        // Get all groups from the server.
        this.subscriptions.add(
            this.serverData.getGroups().subscribe((data) => {
                if (data.items) {
                    this.groups = data.items
                    this.userGroups = [...initUserGroups]
                    for (const i in this.groups) {
                        if (this.groups.hasOwnProperty(i)) {
                            this.userGroups.push({
                                label: this.groups[i].name,
                                value: { id: this.groups[i].id, name: this.groups[i].name },
                            })
                        }
                    }
                }
            })
        )

        // Open the default tab
        this.tabs = [{ label: 'Users', routerLink: '/users/list' }]

        // Store the default tab on the list
        this.openedTabs = []
        const defaultTab = new UserTab(UserTabType.List, null)
        this.openedTabs.push(defaultTab)

        this.subscriptions.add(
            this.route.paramMap.subscribe((params: ParamMap) => {
                const userIdStr = params.get('id')
                if (!userIdStr || userIdStr === 'list') {
                    // Open the tab with the list of users.
                    this.switchToTab(0)
                } else {
                    // Deal with the case when specific user is selected or when the
                    // new user is to be created.
                    const userId = userIdStr === 'new' ? 0 : parseInt(userIdStr, 10)

                    // Iterate over opened tabs and check if any of them matches the
                    // given user id or is for new user.
                    for (const i in this.openedTabs) {
                        if (this.openedTabs.hasOwnProperty(i)) {
                            const tab = this.openedTabs[i]
                            if (
                                (userId > 0 &&
                                    (tab.tabType === UserTabType.User || tab.tabType === UserTabType.EditedUser) &&
                                    tab.user &&
                                    tab.user.id === userId) ||
                                (userId === 0 && tab.tabType === UserTabType.NewUser)
                            ) {
                                this.switchToTab(i)
                                return
                            }
                        }
                    }

                    // If we are creating new user and the tab for the new user does not
                    // exist, let's open the tab and bail.
                    if (userId === 0) {
                        this.showNewUserTab()
                        return
                    }

                    // If we're interested in a tab for a specific user, let's see if we
                    // already have the user information fetched.
                    for (const u of this.users) {
                        if (u.id === userId) {
                            // Found user information, so let's open the tab using this
                            // information and return.
                            this.addUserTab(UserTabType.User, u)
                            return
                        }
                    }

                    // We have no information about the user, so let's try to fetch it
                    // from the server.
                    // ToDo: Non-catches promise
                    lastValueFrom(this.usersApi.getUser(userId)).then((data) => {
                        this.addUserTab(UserTabType.User, data)
                    })
                }
            })
        )
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
            login: this.userForm.controls.userLogin.value,
            email: this.userForm.controls.userEmail.value,
            name: this.userForm.controls.userFirst.value,
            lastname: this.userForm.controls.userLast.value,
            groups: [this.userForm.controls.userGroup.value.id],
            authenticationMethodId: '',
            changePassword: this.userForm.controls.changePassword.value,
        }
        const password = this.userForm.controls.userPassword.value
        const account = { user, password }
        lastValueFrom(this.usersApi.createUser(account))
            .then((/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'New user account created',
                    detail: 'Successfully added new user account.',
                })
                this.closeActiveTab()
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to create new user account',
                    detail: 'Failed to create new user account: ' + msg,
                    sticky: true,
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
            id: this.userTab.user.id,
            login: this.userForm.controls.userLogin.value,
            email: this.userForm.controls.userEmail.value,
            name: this.userForm.controls.userFirst.value,
            lastname: this.userForm.controls.userLast.value,
            groups: [this.userForm.controls.userGroup.value.id],
            authenticationMethodId: '',
            changePassword: this.userForm.controls.changePassword.value,
        }
        const password = this.userForm.controls.userPassword.value
        const account = { user, password }
        lastValueFrom(this.usersApi.updateUser(account))
            .then((/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User account updated',
                    detail: 'Successfully updated user account.',
                })
                this.closeActiveTab()
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to update user account',
                    detail: 'Failed to update user account: ' + msg,
                    sticky: true,
                })
            })
    }

    /*
     * Displays a dialog to confirm user deletion.
     */
    confirmDeleteUser() {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this user?',
            header: 'Delete User',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.deleteUser()
            },
        })
    }

    /**
     * Action invoked when existing user form is being deleted
     *
     * As a result of this action an existing user account is attempted to be
     * deleted.
     */
    deleteUser() {
        lastValueFrom(this.usersApi.deleteUser(this.userTab.user.id))
            .then((/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User account deleted',
                    detail: 'Successfully deleted user account.',
                })
                this.closeActiveTab()
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to delete user account',
                    detail: 'Failed to delete user account: ' + msg,
                    sticky: true,
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
        if (this.newUserTab) {
            this.newUserSave()
        } else if (this.editedUserTab) {
            this.editedUserSave()
        }
    }

    /**
     * Action invoked when Cancel button is clicked under the form
     *
     * It closes current tab with no action.
     */
    userFormCancel() {
        this.closeActiveTab()
    }

    /**
     * Indicates if the user in an active tab is managed by an internal
     * authentication service
     */
    get isInternalUser() {
        const authenticationMethodId = this.userTab.user?.authenticationMethodId
        // Empty or null or internal.
        return !authenticationMethodId || authenticationMethodId === 'internal'
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

        if (this.userTab.userForm.get(name).errors?.['required']) {
            errors.push('This field is required.')
        }

        if (this.userTab.userForm.get(name).errors?.['minlength']) {
            errors.push('This field value is too short.')
        }

        if (this.userTab.userForm.get(name).errors?.['maxlength']) {
            errors.push('This field value is too long.')
        }

        if (this.userTab.userForm.get(name).errors?.['email']) {
            errors.push('Email is invalid.')
        }

        if (this.userTab.userForm.get(name).errors?.['pattern']) {
            errors.push(formatFeedback ?? 'This field value is incorrect.')
        }

        if (comparePasswords && this.userTab.userForm.errors?.['mismatchedPasswords']) {
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
            (this.userTab.userForm.get(name).invalid ||
                (comparePasswords && this.userTab.userForm.errors?.['mismatchedPasswords'])) &&
            (this.userTab.userForm.get(name).dirty || this.userTab.userForm.get(name).touched)
        )
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
}
