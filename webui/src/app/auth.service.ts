import { Injectable } from '@angular/core'
import { Router } from '@angular/router'
import { BehaviorSubject, Observable } from 'rxjs'
import { map, shareReplay, switchMap } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { UsersService } from './backend/api/users.service'
import { AuthenticationMethod } from './backend/model/authenticationMethod'
import { SessionCredentials } from './backend/model/sessionCredentials'
import { User } from './backend'
import { getErrorMessage } from './utils'

@Injectable({
    providedIn: 'root',
})
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>
    public currentUser: Observable<User>
    public user: User
    private authenticationMethods: Observable<AuthenticationMethod[]>

    constructor(
        private api: UsersService,
        private router: Router,
        private msgSrv: MessageService
    ) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')))
        this.currentUser = this.currentUserSubject.asObservable()
        this.authenticationMethods = api.getAuthenticationMethods().pipe(
            map((methods) => methods.items),
            shareReplay(1)
        )
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
     * @param identifier Specified identifier (e.g., user name).
     * @param secret Specified secret (e.g., password).
     * @param authenticationId Specified authentication method ID.
     * @param returnUrl URL to return to after successful login.
     */
    login(authenticationId: string, identifier: string, secret: string, returnUrl: string) {
        let user: User
        const credentials: SessionCredentials = { authenticationId, identifier, secret }
        this.api.createSession(credentials).subscribe(
            (user) => {
                if (user.id != null) {
                    this.currentUserSubject.next(user)
                    localStorage.setItem('currentUser', JSON.stringify(user))
                    // ToDo: Unhandled exception from promise
                    this.router.navigate([returnUrl])
                }
            },
            (err) => {
                const message = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Invalid login or password',
                    detail: message
                })
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

    /**
     * Fetches the list of the supported authentication methods and caches them.
     *
     * @returns List of authentication methods supported by the backend.
     */
    getAuthenticationMethods(): Observable<AuthenticationMethod[]> {
        return this.authenticationMethods
    }
}
