import { HttpErrorResponse, HttpInterceptor, HttpEvent, HttpHandler, HttpRequest } from '@angular/common/http'
import { Router } from '@angular/router'
import { Injectable } from '@angular/core'
import { throwError, of, Observable } from 'rxjs'
import { catchError } from 'rxjs/operators'

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
    constructor(private router: Router) {}

    private handleAuthError(err: HttpErrorResponse): Observable<any> {
        if (err.status === 401) {
            // User is apparently not logged as it got Unauthorized error.
            // Redirect the user to the login page.
            this.router.navigateByUrl('/login')
            return of(err.message)
        }
        return throwError(err)
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).pipe(catchError(x => this.handleAuthError(x)))
    }
}
