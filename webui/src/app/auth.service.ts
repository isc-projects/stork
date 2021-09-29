import { Injectable } from '@angular/core'
import { HttpClient } from '@angular/common/http'
import { Router, ActivatedRoute } from '@angular/router'
import { BehaviorSubject, Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { UsersService } from './backend/api/users.service'

/**
 * Represents credentials of the user who is logging in to the system.
 */
class Credentials {
    useremail: string
    userpassword: string
}

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

    constructor(
        private http: HttpClient,
        private api: UsersService,
        private router: Router,
        private msgSrv: MessageService
    ) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')))
        this.currentUser = this.currentUserSubject.asObservable()
    }

    /**
     * Returns information about currently logged user.
     */
    public get currentUserValue(): User {
        return this.currentUserSubject.value
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
        const credentials: Credentials = { useremail: username, userpassword: password }
        this.api.createSession(credentials).subscribe(
            (data) => {
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
                    // ToDo: Unhandled exception from promise
                    this.router.navigate([returnUrl])
                }
            },
            (err) => {
                this.msgSrv.add({ severity: 'error', summary: 'Invalid login or password' })
            }
        )
        return user
    }

    /**
     * Destroys user session.
     */
    logout() {
        this.api.deleteSession('response').subscribe((resp) => {
            this.destroyLocalSession()
        })
    }

    /**
     * Destroys session information in the local storage.
     */
    destroyLocalSession() {
        localStorage.removeItem('currentUser')
        this.currentUserSubject.next(null)
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
