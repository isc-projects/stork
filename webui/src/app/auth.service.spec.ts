import { TestBed } from '@angular/core/testing'

import { AuthService } from './auth.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { User, UsersService } from './backend'
import { RouterModule, Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'

describe('AuthService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [
                UsersService,
                {
                    provide: Router,
                    useValue: {},
                },
                MessageService,
            ],
            imports: [HttpClientTestingModule, ProgressSpinnerModule],
        })
    )

    it('should be created', () => {
        const service: AuthService = TestBed.inject(AuthService)
        expect(service).toBeTruthy()
    })

    it('should indicate that the user is internal if it uses the internal authentication', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue({
            authenticationMethod: 'internal',
        } as User)
        expect(service.isInternalUser()).toBeTrue()
    })

    it('should indicate that the user is not internal if it uses the external authentication', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue({
            authenticationMethod: 'external',
        } as User)
        expect(service.isInternalUser()).toBeFalse()
    })

    it('should indicate that the user is not internal if the user is not logged', () => {
        const service: AuthService = TestBed.inject(AuthService)
        spyOnProperty(service, 'currentUserValue').and.returnValue(undefined)
        expect(service.isInternalUser()).toBeFalse()
    })
})
