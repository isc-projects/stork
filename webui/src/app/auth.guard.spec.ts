import { TestBed, inject } from '@angular/core/testing'

import { AuthGuard } from './auth.guard'
import { RouterModule, Router } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { UsersService } from './backend'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('AuthGuard', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [RouterModule],
            providers: [
                AuthGuard,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        })
    })

    it('should ...', inject([AuthGuard], (guard: AuthGuard) => {
        expect(guard).toBeTruthy()
    }))
})
