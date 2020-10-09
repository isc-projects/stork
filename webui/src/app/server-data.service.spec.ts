import { TestBed } from '@angular/core/testing'
import { ServerDataService } from './server-data.service'
import { HttpClient, HttpHandler } from '@angular/common/http'

describe('ServerDataService', () => {
    let service: ServerDataService

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(ServerDataService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })
})
