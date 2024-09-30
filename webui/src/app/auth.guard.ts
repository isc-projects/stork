import { Injectable } from '@angular/core'
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree } from '@angular/router'
import { Observable } from 'rxjs'

import { AuthService } from './auth.service'

@Injectable({
    providedIn: 'root',
})
export class AuthGuard {
    isAppInitialized = false
    user: any

    constructor(
        private router: Router,
        private auth: AuthService
    ) {}

    /** Indicates if a user has a permission to activate a given route. */
    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        const currentUser = this.auth.currentUserValue
        if (currentUser) {
            // // check if route is restricted by role
            // if (route.data.roles && route.data.roles.indexOf(currentUser.role) === -1) {
            //     // role not authorized so redirect to home page
            //   this.router.navigate(['/']);
            //     return false;
            // }

            // Check if the user needs to change the password.
            if (currentUser.changePassword && !state.url.startsWith('/profile/password')) {
                // Redirect to the change password page.
                return this.router.createUrlTree(['/profile/password'], { queryParams: { returnUrl: state.url } })
            }

            // authorized so return true
            return true
        }

        // not logged in so redirect to login page with the return url
        return this.router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } })
    }
}
