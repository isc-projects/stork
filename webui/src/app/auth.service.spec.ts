import { TestBed } from '@angular/core/testing'

import { AuthService } from './auth.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { AuthenticationMethods, User, UsersService } from './backend'
import { Router, RouterModule } from '@angular/router'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { from, of } from 'rxjs'
import { HttpProgressEvent } from '@angular/common/http'

describe('AuthService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [UsersService, MessageService],
            imports: [HttpClientTestingModule, ProgressSpinnerModule, RouterModule],
        })
    )

    it('should be created', () => {
        const service: AuthService = TestBed.inject(AuthService)
        expect(service).toBeTruthy()
    })

    it('should indicate that the user is internal if it uses the internal authentication', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue({
            authenticationMethodId: 'internal',
        } as User)
        expect(service.isInternalUser()).toBeTrue()
    })

    it('should indicate that the user is not internal if it uses the external authentication', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue({
            authenticationMethodId: 'external',
        } as User)
        expect(service.isInternalUser()).toBeFalse()
    })

    it('should indicate that the user is not internal if the user is not logged', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue(undefined)
        expect(service.isInternalUser()).toBeFalse()
    })

    it('should fetch the authentication method only once', async () => {
        const usersService: UsersService = TestBed.inject(UsersService)
        const spy = spyOn(usersService, 'getAuthenticationMethods').and.returnValue(
            of({
                total: 1,
                items: [{ id: 'internal' }],
            } as AuthenticationMethods & HttpProgressEvent)
        )

        const authService = TestBed.inject(AuthService)
        const methods1 = await authService.getAuthenticationMethods().toPromise()
        const methods2 = await authService.getAuthenticationMethods().toPromise()

        expect(spy.calls.count()).toBe(1)
        expect(methods1.length).toBe(1)
        expect(methods1[0].id).toBe('internal')
        expect(methods2.length).toBe(1)
        expect(methods2[0].id).toBe('internal')
    })

    it('should fetch the authentication method until success', async () => {
        const usersService: UsersService = TestBed.inject(UsersService)
        const spy = spyOn(usersService, 'getAuthenticationMethods').and.returnValue(
            from([
                // The fetch must be retried until success.
                Error(),
                Error(),
                Error(),
                Error(),
                Error(),
                {
                    total: 1,
                    items: [{ id: 'internal' }],
                } as AuthenticationMethods,
            ]) as any
        )

        const authService = TestBed.inject(AuthService)
        const methods = await authService.getAuthenticationMethods().toPromise()

        expect(spy.calls.count()).toBe(1)
        expect(methods.length).toBe(1)
        expect(methods[0].id).toBe('internal')
    })

    it('should reset the change password flag in the local storage', () => {
        // Arrange
        const service: AuthService = TestBed.inject(AuthService)
        const userService = TestBed.inject(UsersService)
        const router = TestBed.inject(Router)

        spyOn(router, 'navigateByUrl').and.resolveTo(true)
        spyOn(userService, 'createSession').and.returnValue(
            of({
                id: 1,
                changePassword: true,
            } as User) as any
        )

        service.login('internal', 'user', 'password', '/')
        let userFromLocalStorage = JSON.parse(localStorage.getItem('currentUser')) as User

        // Act
        expect(userFromLocalStorage.changePassword).toBeTrue()
        service.resetChangePasswordFlag()

        // Assert
        expect(service.currentUserValue.changePassword).toBeFalse()
        userFromLocalStorage = JSON.parse(localStorage.getItem('currentUser')) as User
        expect(userFromLocalStorage.changePassword).toBeFalse()
    })
})
