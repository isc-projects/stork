import { TestBed } from '@angular/core/testing'

import { AuthService } from './auth.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { UsersService } from './backend'
import { RouterModule, Router } from '@angular/router'
import { MessageService } from 'primeng/api'

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
            imports: [HttpClientTestingModule],
        })
    )

    it('should be created', () => {
        const service: AuthService = TestBed.inject(AuthService)
        expect(service).toBeTruthy()
    })
})
