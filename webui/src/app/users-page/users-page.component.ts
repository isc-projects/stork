import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'
import { MenuItem } from 'primeng/api'

import { UsersService } from '../backend/api/api'

export enum UserTabType {
    List = 1,
    NewUser,
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
    newUserForm: FormGroup

    constructor(private route: ActivatedRoute,
                private router: Router,
                private formBuilder: FormBuilder,
                private usersApi: UsersService) {}

    get existingUserTab(): boolean {
        return this.userTab && this.userTab.tabType === UserTabType.User
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
        this.router.navigate(['/users/:id/edit', 1])
    }

    showNewUserTab() {
        this.newUserForm = this.formBuilder.group({
            userlogin: ['', Validators.required],
            useremail: ['', Validators.required],
            userfirst: ['', Validators.required],
            userlast: ['', Validators.required],
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
                    if (((userId > 0) && (tab.tabType === UserTabType.User) && tab.user &&
                         (tab.user.id === userId)) ||
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
}
