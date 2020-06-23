import { HttpErrorResponse, HttpInterceptor, HttpEvent, HttpHandler, HttpRequest } from '@angular/common/http'
import { Router } from '@angular/router'
import { Injectable } from '@angular/core'
import { throwError, of, Observable } from 'rxjs'
import { catchError } from 'rxjs/operators'

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
    constructor(private router: Router) {}

    private handleAuthError(err: HttpErrorResponse): Observable<any> {
        switch (err.status) {
            case 401:
                // User is apparently not logged in as it got Unauthorized error.
                // Redirect the user to the login page.
                this.router.navigateByUrl('/login')
                return of(err.message)
            case 403:
                // User has no access to the given view. Let's redirect the
                // user to the error page.
                this.router.navigateByUrl('/forbidden', { skipLocationChange: true })
                return of(err.message)
            default:
                return throwError(err)
        }
        return throwError(err)
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).pipe(catchError((x) => this.handleAuthError(x)))
    }
}
