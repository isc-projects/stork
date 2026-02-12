import { TestBed } from '@angular/core/testing'

import { AuthService, isInternalUser } from './auth.service'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { AuthenticationMethods, User, UsersService } from './backend'
import { Router, provideRouter } from '@angular/router'
import { MessageService } from 'primeng/api'
import { from, of } from 'rxjs'
import { HttpProgressEvent, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('AuthService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        })
    )

    it('should be created', () => {
        const service: AuthService = TestBed.inject(AuthService)
        expect(service).toBeTruthy()
    })

    it('should indicate that the user is internal if it uses the internal authentication', () => {
        const user: User = { id: 1, authenticationMethodId: 'internal' }
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue(user)
        expect(service.isInternalUser()).toBeTrue()
        expect(isInternalUser(user)).toBeTrue()
    })

    it('should indicate that the user is not internal if it uses the external authentication', () => {
        const user: User = { id: 1, authenticationMethodId: 'external' }
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue(user)
        expect(service.isInternalUser()).toBeFalse()
        expect(isInternalUser(user)).toBeFalse()
    })

    it('should indicate that the user is not internal if the user is not logged', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue(undefined)
        expect(service.isInternalUser()).toBeFalse()
        expect(isInternalUser(undefined)).toBeFalse()
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

    it('should reset the change password flag for currently authenticated user', () => {
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

        // Act
        service.resetChangePasswordFlag()

        // Assert
        expect(service.currentUserValue.changePassword).toBeFalse()
    })
})
