import { TestBed } from '@angular/core/testing'

import { AuthService } from './auth.service'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { UsersService } from './backend'
import { RouterModule, Router } from '@angular/router'
import { MessageService } from 'primeng/api'

describe('AuthService', () => {
    beforeEach(() => TestBed.configureTestingModule({
        providers: [HttpClient, HttpHandler, UsersService, {
            provide: Router, useValue: {}
        }, MessageService]
    }))

    it('should be created', () => {
        const service: AuthService = TestBed.inject(AuthService)
        expect(service).toBeTruthy()
    })
})
