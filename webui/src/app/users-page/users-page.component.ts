import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormControl, FormGroup, NgForm, Validators } from '@angular/forms'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'
import { MenuItem, MessageService } from 'primeng/api'

import { UsersService } from '../backend/api/api'
import { UserAccount } from '../backend/model/models'

export enum UserTabType {
    List = 1,
    NewUser,
    EditedUser,
    User,
}

export class UserTab {
    constructor(public tabType: UserTabType, public user: any) { }

    get tabRoute(): string {
        switch (this.tabType) {
            case UserTabType.List: {
                return "/users/list"
            }
            case UserTabType.NewUser: {
                return "/users/new"
            }
            default: {
                if (this.user) {
                    return "/users/" + this.user.id
                }
            }
        }
        return "/users/list"
    }
}

function matchPasswords(passwordKey: string, confirmPasswordKey: string) {
    return (group: FormGroup): {[key: string]: any} => {
        let password = group.controls[passwordKey];
        let confirmPassword = group.controls[confirmPasswordKey];

        if (password.value !== confirmPassword.value) {
            return {
                mismatchedPasswords: true
            };
        }
    }
}

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

    // form
    public userform: FormGroup

    constructor(private route: ActivatedRoute,
                private router: Router,
                private formBuilder: FormBuilder,
                private usersApi: UsersService,
                private msgSrv: MessageService) {}

    get existingUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.User
    }

    get editedUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.EditedUser
    }

    get newUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.NewUser
    }

    switchToTab(index) {
        if (this.activeTabIdx === index) {
            return
        }
        this.activeTabIdx = index
        this.activeItem = this.tabs[index]
        if (index > 0) {
            this.userTab = this.openedTabs[index]
            this.router.navigate([this.userTab.tabRoute])
        } else {
            this.userTab = null
            this.router.navigate(['/users/list'])
        }
    }

    addUserTab(tabType: UserTabType, user) {
        let userTab = new UserTab(tabType, user)
        this.openedTabs.push(userTab)
        this.userTab = userTab
        this.tabs.push({
            label: (tabType == UserTabType.NewUser ? "new account" : user.login || user.email),
            routerLink: userTab.tabRoute
        })
    }

    editUserInfo(tab) {
        this.userform = this.formBuilder.group({
            userlogin: ['', Validators.required],
            useremail: ['', Validators.email],
            userfirst: ['', Validators.required],
            userlast: ['', Validators.required],
            userpassword: ['', Validators.minLength(8)],
            userpassword2: ['', Validators.minLength(8)],
        }, {validators: [matchPasswords('userpassword', 'userpassword2')]})

        tab.tabType = UserTabType.EditedUser
        this.userform.patchValue({
            userlogin: this.userTab.user.login,
            useremail: this.userTab.user.email,
            userfirst: this.userTab.user.name,
            userlast: this.userTab.user.lastname,
        })
    }

    showNewUserTab() {
        this.userform = this.formBuilder.group({
            userlogin: ['', Validators.required],
            useremail: ['', Validators.email],
            userfirst: ['', Validators.required],
            userlast: ['', Validators.required],
            userpassword: ['', Validators.compose([Validators.required, Validators.minLength(8)])],
            userpassword2: ['', Validators.required],
        })

        for (let i in this.openedTabs) {
            if (this.openedTabs[i].tabType === UserTabType.NewUser) {
                this.switchToTab(i)
                return
            }
        }
        this.addUserTab(UserTabType.NewUser, null)
        this.switchToTab(this.tabs.length - 1)
    }

    ngOnInit() {
        this.tabs = [{ label: 'Users', routerLink: '/users/list' }]

        let defaultTab = new UserTab(UserTabType.List, null)
        this.openedTabs = [ ]
        this.openedTabs.push(defaultTab)

        this.users = []

        this.route.paramMap.subscribe((params: ParamMap) => {
            const userIdStr = params.get('id')
            if (userIdStr === 'list') {
                this.switchToTab(0)

            } else {
                const userId = (userIdStr === 'new') ? 0 : parseInt(userIdStr, 10)

                let found = false
                for (let i in this.openedTabs) {
                    let tab = this.openedTabs[i]
                    if (((userId > 0) &&
                         ((tab.tabType === UserTabType.User) ||
                          (tab.tabType === UserTabType.EditedUser)) &&
                         tab.user && (tab.user.id === userId)) ||
                        ((userId == 0) && tab.tabType === UserTabType.NewUser)) {
                        this.switchToTab(i)
                        found = true
                        break

                    }
                }

                if (!found) {
                    for (const u of this.users) {
                        if (u.id === userId) {
                            this.addUserTab(UserTabType.User, u)
                            this.switchToTab(this.tabs.length - 1)
                        }
                    }
                }
            }
        })
    }

    loadUsers(event) {
        this.usersApi.getUsers(event.first, event.rows, event.filters.text).subscribe(data => {
            this.users = data.items
            this.totalUsers = data.total
        })
    }

    closeActiveTab() {
        this.openedTabs.splice(this.activeTabIdx, 1)
        this.tabs.splice(this.activeTabIdx, 1)
        this.switchToTab(0)
    }

    closeTab(event, idx) {
        this.openedTabs.splice(idx, 1)
        this.tabs.splice(idx, 1)
        if (this.activeTabIdx == idx) {
            this.switchToTab(idx - 1)

        } else if (this.activeTabIdx > idx) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    newUserSave() {
        const user = {
            id: 0,
            login: this.userform.controls.userlogin.value,
            email: this.userform.controls.useremail.value,
            name: this.userform.controls.userfirst.value,
            lastname: this.userform.controls.userlast.value,
        }
        const password = this.userform.controls.userpassword.value
        const account = { user: user, password: password }
        this.usersApi.createUser(account).subscribe(data => {
            console.info(data)
            this.msgSrv.add({
                severity: 'success',
                summary: 'New user account created',
                detail: 'Adding new user account succeeeded',
            })
            this.closeActiveTab()
            err => {
                console.info(err)
                let msg = err.StatusText
                if (err.error && err.error.detail) {
                    msg = err.error.detail
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Creating new user account failed',
                    detail: 'Creating new user account failed: ' + msg,
                    sticky: true,
                })
            }
        })
    }

    editedUserSave() {
        const user = {
            id: this.userTab.user.id,
            login: this.userform.controls.userlogin.value,
            email: this.userform.controls.useremail.value,
            name: this.userform.controls.userfirst.value,
            lastname: this.userform.controls.userlast.value,
        }
        const password = this.userform.controls.userpassword.value
        const account = { user: user, password: password }
    }

    userFormSave() {
        if (this.newUserTab) {
            this.newUserSave()
        } else if (this.editedUserTab) {
            this.editedUserSave()
        }
    }
}
