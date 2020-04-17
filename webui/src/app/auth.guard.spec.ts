import { TestBed, async, inject } from '@angular/core/testing'

import { AuthGuard } from './auth.guard'
import { RouterModule, Router } from '@angular/router'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { UsersService } from './backend'
import { MessageService } from 'primeng/api'

describe('AuthGuard', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [RouterModule],
            providers: [AuthGuard, HttpClient, HttpHandler, UsersService, MessageService, {
                provide: Router,
                useValue: {}
            }
            ]
        })
    })

    it('should ...', inject([AuthGuard], (guard: AuthGuard) => {
        expect(guard).toBeTruthy()
    }))
})
