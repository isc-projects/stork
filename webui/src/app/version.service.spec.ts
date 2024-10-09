import { TestBed } from '@angular/core/testing'

import { VersionService } from './version.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'

describe('VersionService', () => {
    let service: VersionService

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [],
            imports: [HttpClientTestingModule],
        })
        service = TestBed.inject(VersionService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })
})
