import { Injectable } from '@angular/core'
import { HttpClient } from '@angular/common/http'
import { Router, ActivatedRoute } from '@angular/router'
import { BehaviorSubject, Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { AppInitService } from './app-init.service'
import { UsersService } from './backend/api/users.service'

export class User {
    id: number
    username: string
    email: string
    firstName: string
    lastName: string
    groups: number[]
}

/**
 * Represents system group fetched from the database.
 */
export class SystemGroup {
    id: number
    name: string
    description: string
}

@Injectable({
    providedIn: 'root',
})
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>
    public currentUser: Observable<User>
    public user: User
    public groups: SystemGroup[]

    constructor(
        private http: HttpClient,
        private api: UsersService,
        private router: Router,
        private msgSrv: MessageService,
        private appInit: AppInitService
    ) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')))
        this.currentUser = this.currentUserSubject.asObservable()
        this.initSystemGroups()
    }

    /**
     * Returns information about currently logged user.
     */
    public get currentUserValue(): User {
        return this.currentUserSubject.value
    }

    /**
     * Returns name of the system group fetched from the database.
     *
     * @param groupId Identifier of the group in the database, counted
     *                from 1.
     * @returns Group name or unknown string if the group is not found.
     */
    public groupName(groupId): string {
        const groupIdx = groupId - 1
        if (this.groups.hasOwnProperty(groupIdx)) {
            return this.groups[groupIdx].name
        }
        return 'unknown'
    }

    /**
     * Initializes a list of system groups.
     *
     * The groups are fetched from the database by the {@link AppInitService}
     * prior to page load.
     */
    initSystemGroups() {
        this.groups = []
        const groups = this.appInit.groups
        if (groups) {
            for (const i in groups) {
                if (groups.hasOwnProperty(i)) {
                    const group = new SystemGroup()
                    group.id = groups[i].id
                    group.name = groups[i].name
                    group.description = groups[i].description
                    this.groups.push(group)
                }
            }
        }
    }

    /**
     * Attempts to create a session for a user.
     *
     * @param username Specified user name.
     * @param password Specified password.
     * @param returnUrl URL to return to after successful login.
     */
    login(username: string, password: string, returnUrl: string) {
        let user: User
        this.api.createSession(username, password).subscribe(
            data => {
                if (data.id != null) {
                    user = new User()

                    user.id = data.id
                    user.username = data.login
                    user.email = data.email
                    user.firstName = data.name
                    user.lastName = data.lastname

                    // Store groups the user belongs to.
                    user.groups = []
                    for (const i in data.groups) {
                        if (data.groups.hasOwnProperty(i)) {
                            user.groups.push(data.groups[i])
                        }
                    }

                    this.currentUserSubject.next(user)
                    localStorage.setItem('currentUser', JSON.stringify(user))
                    this.router.navigate([returnUrl])
                }
            },
            err => {
                this.msgSrv.add({ severity: 'error', summary: 'Invalid login or password' })
            }
        )
        return user
    }

    /**
     * Destroys user session.
     */
    logout() {
        this.api.deleteSession('response').subscribe(resp => {
            localStorage.removeItem('currentUser')
            this.currentUserSubject.next(null)
        })
    }

    /**
     * Convenience function checking if the current user has the super admin role.
     *
     * @returns true if the user has super-admin group.
     */
    superAdmin(): boolean {
        if (this.currentUserValue && this.currentUserValue.groups) {
            for (const i in this.currentUserValue.groups) {
                if (this.currentUserValue.groups[i] === 1) {
                    return true
                }
            }
        }
        return false
    }
}
