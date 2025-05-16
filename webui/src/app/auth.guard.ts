import { Injectable } from '@angular/core'
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree } from '@angular/router'
import { Observable, of } from 'rxjs'

import { AccessType, AuthService, PrivilegeKey } from './auth.service'
import { mergeMap } from 'rxjs/operators'

@Injectable({
    providedIn: 'root',
})
export class AuthGuard {
    constructor(
        private router: Router,
        private auth: AuthService
    ) {}

    /** Indicates if a user has a permission to activate a given route. */
    canActivate(
        _route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        // Below is an async canActivate implementation. First we must retrieve an information from the auth service
        // whether the user is authenticated. Auth service getUserOrRetrieveFromSession() observable will:
        // 1. emit user object if already authenticated
        // 2. if not, will try to retrieve the user object from the session in backend and emit the user object
        // 3. if not retrieved in 2, will emit null - which means that user is not authenticated.
        // The observable does not emit errors, only user object or null.
        return this.auth.getUserOrRetrieveFromSession().pipe(
            mergeMap((currentUser) => {
                if (!currentUser) {
                    // not logged in so redirect to login page with the return url
                    return of(this.router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } }))
                }

                // Stork authorization checks.
                if (
                    route.routeConfig?.data?.key &&
                    !this.auth.hasPrivilege(
                        route.routeConfig.data.key as PrivilegeKey,
                        (route.routeConfig.data?.accessType as AccessType) ?? 'read'
                    )
                ) {
                    // If user is not authorized to access the path, redirect to Forbidden page.
                    return of(this.router.parseUrl('/forbidden'))
                }

                // Check if the user needs to change the password.
                if (currentUser.changePassword && !state.url.startsWith('/profile/password')) {
                    // Redirect to the change password page.
                    return of(
                        this.router.createUrlTree(['/profile/password'], { queryParams: { returnUrl: state.url } })
                    )
                }

                // authorized so return true
                return of(true)
            })
        )
    }
}
