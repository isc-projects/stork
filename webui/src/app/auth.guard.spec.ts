import { TestBed, inject } from '@angular/core/testing'

import { AuthGuard } from './auth.guard'
import { RouterModule, Router } from '@angular/router'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { UsersService } from './backend'
import { MessageService } from 'primeng/api'

describe('AuthGuard', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [RouterModule, HttpClientTestingModule],
            providers: [
                AuthGuard,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
            ],
        })
    })

    it('should ...', inject([AuthGuard], (guard: AuthGuard) => {
        expect(guard).toBeTruthy()
    }))
})
