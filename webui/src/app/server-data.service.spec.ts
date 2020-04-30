import { TestBed } from '@angular/core/testing'

import { ServerDataService } from './server-data.service'

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
