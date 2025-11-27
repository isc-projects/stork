import { TestBed, inject } from '@angular/core/testing'

import { AuthGuard } from './auth.guard'
import { Router, provideRouter } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('AuthGuard', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        })
    })

    it('should ...', inject([AuthGuard], (guard: AuthGuard) => {
        expect(guard).toBeTruthy()
    }))
})
