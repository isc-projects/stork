import { TestBed } from '@angular/core/testing'
import { ServerDataService } from './server-data.service'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { Router } from '@angular/router'
import { ServicesService, UsersService } from './backend'
import { MessageService } from 'primeng/api'

describe('ServerDataService', () => {
    let service: ServerDataService

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [HttpClient, HttpHandler, UsersService, MessageService, ServicesService,
                {
                    provide: Router,
                    useValue: {}
                }]
        })
        service = TestBed.inject(ServerDataService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })
})
