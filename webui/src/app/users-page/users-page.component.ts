import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormControl, FormGroup, NgForm, Validators } from '@angular/forms'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'
import { MenuItem, MessageService, SelectItem } from 'primeng/api'

import { AuthService } from '../auth.service'
import { UsersService } from '../backend/api/api'
import { UserAccount } from '../backend/model/models'

/**
 * An enum specifying tab types in the user view
 *
 * Currently supported types are:
 * - list: including a list of users
 * - new user: including a form for creating new user account
 * - edited user: including a form for editing user account
 * - user: including read only information about the user
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
    public userform: FormGroup

    /**
     * Constructor
     */
    constructor(public tabType: UserTabType, public user: any) {}

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
 *                           password can be fiund in the form.
 * @returns The validator function comparing the passwords.
 */
function matchPasswords(passwordKey: string, confirmPasswordKey: string) {
    return (group: FormGroup): { [key: string]: any } => {
        const password = group.controls[passwordKey]
        const confirmPassword = group.controls[confirmPasswordKey]

        if (password.value !== confirmPassword.value) {
            return {
                mismatchedPasswords: true,
            }
        }
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
export class UsersPageComponent implements OnInit {
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

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: FormBuilder,
        private usersApi: UsersService,
        private msgSrv: MessageService,
        public auth: AuthService
    ) {}

    /**
     * Returns user form from the current tab.
     *
     * @returns instance of the form or null if the curren tab includes no form.
     */
    get userform(): FormGroup {
        return this.userTab ? this.userTab.userform : null
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
        this.tabs.push({
            label: tabType === UserTabType.NewUser ? 'new account' : user.login || user.email,
            routerLink: userTab.tabRoute,
        })
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
        const userform = this.formBuilder.group(
            {
                userlogin: ['', Validators.required],
                useremail: ['', Validators.email],
                userfirst: ['', Validators.required],
                userlast: ['', Validators.required],
                usergroup: ['', Validators.required],
                userpassword: ['', Validators.minLength(8)],
                userpassword2: ['', Validators.minLength(8)],
            },
            { validators: [matchPasswords('userpassword', 'userpassword2')] }
        )

        // Modify the current tab type to 'edit'.
        tab.tabType = UserTabType.EditedUser

        // Set default values for the fields which may be edited by the user.
        userform.patchValue({
            userlogin: this.userTab.user.login,
            useremail: this.userTab.user.email,
            userfirst: this.userTab.user.name,
            userlast: this.userTab.user.lastname,
        })

        if (
            this.auth.groups &&
            this.auth.groups.length > 0 &&
            this.userTab.user.groups &&
            this.userTab.user.groups.length > 0
        ) {
            userform.patchValue({
                usergroup: {
                    id: this.auth.groups[this.userTab.user.groups[0] - 1].id,
                    name: this.auth.groups[this.userTab.user.groups[0] - 1].name,
                },
            })
        }

        this.userTab.userform = userform
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
        const userform = this.formBuilder.group({
            userlogin: ['', Validators.required],
            useremail: ['', Validators.email],
            userfirst: ['', Validators.required],
            userlast: ['', Validators.required],
            usergroup: ['', Validators.required],
            userpassword: ['', Validators.compose([Validators.required, Validators.minLength(8)])],
            userpassword2: ['', Validators.required],
        })

        // Search opened tabs for the 'new account' type.
        for (const i in this.openedTabs) {
            if (this.openedTabs[i].tabType === UserTabType.NewUser) {
                // The tab exists, simply activate it.
                this.switchToTab(i)
                return
            }
        }
        // The tab doesn't exist, so open it and activate it.
        this.addUserTab(UserTabType.NewUser, null)
        this.userTab.userform = userform
    }

    /**
     * Closes current tab
     */
    private closeActiveTab() {
        this.openedTabs.splice(this.activeTabIdx, 1)
        this.tabs.splice(this.activeTabIdx, 1)
        this.switchToTab(0)
    }

    /**
     * Closes selected tab
     */
    closeTab(event, idx) {
        const i = Number(idx)

        this.openedTabs.splice(i, 1)
        this.tabs.splice(i, 1)
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
        this.usersApi.getUsers(event.first, event.rows, event.filters.text).subscribe(
            data => {
                this.users = data.items
                this.totalUsers = data.total
            },
            err => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Loading user accounts failed',
                    detail: 'Loading user accounts from the database failed: ' + msg,
                    sticky: true,
                })
            }
        )
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
        this.userGroups = [
            {
                label: 'Select Group',
                value: null,
            },
        ]
        for (const i in this.auth.groups) {
            if (this.auth.groups.hasOwnProperty(i)) {
                this.userGroups.push({
                    label: this.auth.groups[i].name,
                    value: { id: this.auth.groups[i].id, name: this.auth.groups[i].name },
                })
            }
        }

        // Open the default tab
        this.tabs = [{ label: 'Users', routerLink: '/users/list' }]

        // Store the default tab on the list
        this.openedTabs = []
        const defaultTab = new UserTab(UserTabType.List, null)
        this.openedTabs.push(defaultTab)

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
                this.usersApi.getUser(userId).subscribe(data => {
                    this.addUserTab(UserTabType.User, data)
                })
            }
        })
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
            login: this.userform.controls.userlogin.value,
            email: this.userform.controls.useremail.value,
            name: this.userform.controls.userfirst.value,
            lastname: this.userform.controls.userlast.value,
            groups: [this.userform.controls.usergroup.value.id],
        }
        const password = this.userform.controls.userpassword.value
        const account = { user, password }
        this.usersApi.createUser(account).subscribe(
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'New user account created',
                    detail: 'Adding new user account succeeeded',
                })
                this.closeActiveTab()
            },
            err => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Creating new user account failed',
                    detail: 'Creating new user account failed: ' + msg,
                    sticky: true,
                })
            }
        )
    }

    /**
     * Action invoked when the form for editing the user information is saved
     *
     * As a result of this action, the user account information will be updated.
     */
    private editedUserSave() {
        const user = {
            id: this.userTab.user.id,
            login: this.userform.controls.userlogin.value,
            email: this.userform.controls.useremail.value,
            name: this.userform.controls.userfirst.value,
            lastname: this.userform.controls.userlast.value,
            groups: [this.userform.controls.usergroup.value.id],
        }
        const password = this.userform.controls.userpassword.value
        const account = { user, password }

        this.usersApi.updateUser(account).subscribe(
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User account updated',
                    detail: 'Updating user account succeeeded',
                })
                this.closeActiveTab()
            },
            err => {
                console.info(err)
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Updating user account failed',
                    detail: 'Updating user account failed: ' + msg,
                    sticky: true,
                })
            }
        )
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
}
