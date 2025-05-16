import { Injectable } from '@angular/core'
import { Router } from '@angular/router'
import { BehaviorSubject, defer, Observable, of, tap, timeout, timer } from 'rxjs'
import { catchError, map, mergeMap, retry, share, shareReplay } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { UsersService } from './backend'
import { AuthenticationMethod } from './backend'
import { SessionCredentials } from './backend'
import { User } from './backend'
import { getErrorMessage } from './utils'

/**
 * System user groups enumeration.
 * IDs match group IDs in the backend.
 */
export enum UserGroup {
    SuperAdmin = 1,
    Admin,
    ReadOnly,
}

/**
 * This type gathers entities that are subject to authorization checks.
 * The entities relate to REST API endpoints, UI components, route paths.
 * In the future the entities may be located in the backend DB together with
 * granted privileges for users and group of users.
 * Type was used instead of Enum, as Enums usage is troublesome in Angular HTML templates.
 */
export type ManagedAccessEntity =
    | 'machines-server-token'
    | 'kea-config-hashes'
    | 'app'
    | 'global-config-checkers'
    | 'daemon-config-checkers'
    | 'host-reservation'
    | 'kea-global-parameters-transaction'
    | 'stork-settings'
    | 'shared-network'
    | 'subnet'
    | 'zones'
    | 'app-access-point-key'
    | 'machine-address'
    | 'machine-authorization'
    | 'machine'
    | 'daemon-config-review'
    | 'daemon-global-config'
    | 'daemon-config'
    | 'communication'
    | 'daemon-monitoring'
    | 'leases'
    | 'swagger'
    | 'all-users'
    | 'specific-user'
    | 'user-password'
    | 'json-config-secret'
    | 'events'
    | 'logs'
    | 'versions'

/**
 * Possible access types in authorization.
 * Naming convention relates to CRUD operations.
 */
export type AccessType = 'create' | 'read' | 'update' | 'delete'

@Injectable({
    providedIn: 'root',
})
export class AuthService {
    /**
     * RxJS behavior subject holding currently authenticated user. In case no user is authenticated, it holds null.
     * @private
     */
    private _currentUserSubject: BehaviorSubject<User>

    /**
     * RxJS observable stream emitting either User or null whenever user gets authenticated or logged out.
     */
    public currentUser$: Observable<User>

    /**
     * RxJS observable used to retrieve from backend and cache authentication methods.
     * @private
     */
    private readonly _authenticationMethods: Observable<AuthenticationMethod[]>

    /**
     * Returns RxJS observable which first tries to emit currently authenticated user.
     * In case no user is authenticated, it tries to retrieve the user from the session in backend.
     * This is useful when user closed the browser and tries to reopen Stork page again.
     * Since Stork keeps persistent session cookie, it will send the request signed with the session token.
     * If the session is alive, backend will send back authenticated user object. Otherwise, it will send
     * back 404, which means that the user will have to authenticate again.
     * In case of any error or timeout, the observable emits null, which also means
     * that the user will have to authenticate again.
     */
    public getUserOrRetrieveFromSession(): Observable<User> {
        return defer(() => of(this.currentUserValue)).pipe(
            mergeMap((u) => {
                if (u) {
                    // This is normal operation when user is authenticated and browser was not closed.
                    // User object is emitted from the _currentUserSubject.
                    return of(u)
                }

                // No authenticated user, so let's check if it can be retrieved from backend.
                // TODO: Check if session cookie exists. If not, there is no point in sending this request.
                return this.api.getSession().pipe(
                    tap((u: User) => {
                        if (u) {
                            this._currentUserSubject.next(u)
                        }
                    }),
                    // Let's not wait too long and gracefully handle the timeout below.
                    timeout(1000),
                    // Let's gracefully handle all errors so that they are not propagated to the subscribers.
                    catchError((err) => {
                        // 404 is normal endpoint response meaning that there is no session alive for the user.
                        if (err.status != 404) {
                            const msg = getErrorMessage(err)
                            this.msgSrv.add({
                                severity: 'error',
                                summary: 'Failed to retrieve session',
                                detail: 'Failed to retrieve session: ' + msg,
                            })
                        }

                        this.destroyAuthenticatedUser()
                        return of(null)
                    }),
                    // In case there are more subscribers, share the response.
                    share()
                )
            }),
            // In case there are more subscribers, share the response.
            share()
        )
    }

    constructor(
        private api: UsersService,
        private router: Router,
        private msgSrv: MessageService
    ) {
        this._currentUserSubject = new BehaviorSubject<User>(null)
        this.currentUser$ = this._currentUserSubject.asObservable()
        this._authenticationMethods = api.getAuthenticationMethods().pipe(
            // Delay to limit the number of requests sent to the backend on
            // failure. Waits in sequence 1, 2, 4, 8, 16, and max. 32 seconds.
            retry({ delay: (_, count) => timer(1000 * 2 ** Math.min(count, 5)) }),
            map((methods) => methods.items),
            shareReplay(1)
        )
    }

    /**
     * Returns information about currently logged user.
     */
    public get currentUserValue(): User {
        return this._currentUserSubject.value
    }

    /**
     * Attempts to create a session for a user.
     *
     * @param identifier Specified identifier (e.g., user name).
     * @param secret Specified secret (e.g., password).
     * @param authenticationMethodId Specified authentication method ID.
     * @param returnUrl URL to return to after successful login.
     */
    login(authenticationMethodId: string, identifier: string, secret: string, returnUrl: string) {
        const credentials: SessionCredentials = { authenticationMethodId, identifier, secret }
        this.api
            .createSession(credentials)
            .pipe(shareReplay())
            .subscribe({
                next: (user) => {
                    if (user.id != null) {
                        // TODO: retrieve expiry date of the session cookie so that it can be used in the UI.
                        this._currentUserSubject.next(user)
                        // ToDo: Unhandled exception from promise
                        this.router.navigateByUrl(returnUrl)
                    }
                },
                error: () => this.msgSrv.add({ severity: 'error', summary: 'Invalid login or password' }),
            })
    }

    /**
     * Destroys user session.
     */
    logout() {
        this.api
            .deleteSession('response')
            .pipe(shareReplay())
            .subscribe({
                next: (/* resp */) => {
                    this.destroyAuthenticatedUser()
                },
                error: (err) => {
                    const msg = getErrorMessage(err)
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Failed to logout user',
                        detail: 'Failed to logout user ' + msg,
                    })
                },
            })
    }

    /**
     * Destroys information about currently authenticated user.
     */
    destroyAuthenticatedUser() {
        this._currentUserSubject.next(null)
    }

    /**
     * Convenience function checking if the current user has the super admin role.
     *
     * @returns true if the user has super-admin group.
     */
    superAdmin(): boolean {
        if (this.currentUserValue && this.currentUserValue.groups) {
            for (const i in this.currentUserValue.groups) {
                if (this.currentUserValue.groups[i] === UserGroup.SuperAdmin) {
                    return true
                }
            }
        }
        return false
    }

    /**
     * Convenience function checking if the current user was authenticated
     * using the credentials stored in the Stork database.
     *
     * @returns true if the user was authenticated using the internal method.
     */
    isInternalUser(): boolean {
        return this.currentUserValue?.authenticationMethodId === 'internal'
    }

    /**
     * Fetches the list of the supported authentication methods and caches them.
     *
     * @returns List of authentication methods supported by the backend.
     */
    getAuthenticationMethods(): Observable<AuthenticationMethod[]> {
        return this._authenticationMethods
    }

    /**
     * Resets the change password flag for the current user.
     */
    resetChangePasswordFlag() {
        const user: User = { ...this.currentUserValue, changePassword: false }
        this._currentUserSubject.next(user)
    }

    /**
     * Returns whether current user has given privilege for given component.
     * @param entityKey component for which the privilege is required
     * @param accessType access type which is required
     */
    hasPrivilege(entityKey: ManagedAccessEntity, accessType: AccessType = 'read'): boolean {
        // For now all privileges are checked based on group that user belongs to.
        // TODO: Privileges should be retrieved from backend when user gets authenticated. All privileges should be destroyed at logout.
        if (this.superAdmin()) {
            // User that belongs to SuperAdmin group, has all privileges.
            return true
        } else if (this.isAdmin()) {
            switch (entityKey) {
                case 'app-access-point-key': // Admin role is not enough to see Access Point Key (it is secret).
                case 'machines-server-token': // Admin role is not enough to see server token (it is secret).
                case 'machine-authorization': // Admin role is not enough to authorize or unauthorize machine.
                case 'all-users': // Admin role can't even read all users.
                case 'json-config-secret': // Admin role is not enough to see secrets in configs.
                    return false
                case 'specific-user':
                    return accessType === 'read' // Admin group can only read their own user data.
                case 'machine':
                    return accessType !== 'delete' // Admin group can't delete machines.
                default:
                    return true
            }
        } else if (this.isInReadOnlyGroup()) {
            switch (entityKey) {
                case 'app-access-point-key':
                case 'machines-server-token':
                case 'all-users':
                case 'json-config-secret':
                    return false
                case 'user-password':
                    return true
                default:
                    return accessType === 'read'
            }
        }

        return false
    }

    /**
     * Returns whether current user belongs to Admin group.
     */
    isAdmin(): boolean {
        for (const i in this.currentUserValue?.groups ?? []) {
            if (this.currentUserValue.groups[i] === UserGroup.Admin) {
                return true
            }
        }

        return false
    }

    /**
     * Returns whether current user belongs to ReadOnly group.
     */
    isInReadOnlyGroup(): boolean {
        for (const i in this.currentUserValue?.groups ?? []) {
            if (this.currentUserValue.groups[i] === UserGroup.ReadOnly) {
                return true
            }
        }

        return false
    }
}
